// iinternal/vecX/sdk.go

package vecX

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/qdrant/go-client/qdrant"
	ort "github.com/yalue/onnxruntime_go"
)

type Service interface {
	Encode(text string) ([]float32, error)
	// AddVector 现在可以看作 UpsertVector 的一个别名
	AddVector(ctx context.Context, id string, vector []float32, payload map[string]interface{}) error
	SearchSimilarVectors(ctx context.Context, queryVector []float32, topK uint64) ([]*pb.ScoredPoint, error)
	DeleteVectors(ctx context.Context, ids []string) error // <-- 新增
	Close()
}

// serviceImpl 是接口的内部实现
type serviceImpl struct {
	tokenizerService  *tokenizer
	sess              *ort.DynamicAdvancedSession
	collectionsClient pb.CollectionsClient
	vectorDBClient    pb.PointsClient
	qdrantConn        *grpc.ClientConn
}

// tokenizer 和 TokenizerResponse 结构体无变化
type tokenizer struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
}
type TokenizerResponse struct {
	InputIDs      [][]int64 `json:"input_ids"`
	AttentionMask [][]int64 `json:"attention_mask"`
}

const (
	qdrantAddress        = "localhost:6334"
	vectorCollectionName = "agent_memory"
	vectorSize           = 768
)

// New 函数无变化
func New() (Service, error) {
	svc := &serviceImpl{}

	log.Println("--- VecX SDK 初始化开始 ---")

	_, currentFilePath, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("无法获取 SDK 文件路径")
	}
	sdkRoot := filepath.Dir(currentFilePath)

	modelLoadStart := time.Now()
	ort.SetSharedLibraryPath(filepath.Join(sdkRoot, "libonnxruntime.dylib"))
	err := ort.InitializeEnvironment()
	if err != nil {
		return nil, fmt.Errorf("无法初始化 ONNX Runtime 环境: %w", err)
	}
	inputNames := []string{"input_ids", "attention_mask"}
	outputNames := []string{"last_hidden_state"}
	modelPath := filepath.Join(sdkRoot, "mpnet_onnx", "model.onnx")
	svc.sess, err = ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("无法创建 ONNX 会话: %w", err)
	}
	log.Printf("模型加载成功！耗时: %s", time.Since(modelLoadStart))

	tokenizerLoadStart := time.Now()
	svc.tokenizerService = new(tokenizer)
	pythonPath := filepath.Join(sdkRoot, ".venv", "bin", "python3")
	scriptPath := filepath.Join(sdkRoot, "tokenizer", "tokenizer.py")
	svc.tokenizerService.cmd = exec.Command(pythonPath, scriptPath)
	svc.tokenizerService.cmd.Dir = sdkRoot
	svc.tokenizerService.cmd.Stderr = os.Stderr
	svc.tokenizerService.stdin, err = svc.tokenizerService.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := svc.tokenizerService.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	svc.tokenizerService.stdout = bufio.NewScanner(stdoutPipe)
	if err := svc.tokenizerService.cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 tokenizer 服务失败: %w", err)
	}
	log.Printf("Tokenizer 子进程已启动，正在等待其完全加载...")
	if svc.tokenizerService.stdout.Scan() {
		if svc.tokenizerService.stdout.Text() != "TOKENIZER_READY" {
			return nil, fmt.Errorf("从 tokenizer 服务收到了意外的启动信号: %s", svc.tokenizerService.stdout.Text())
		}
	} else {
		if err := svc.tokenizerService.stdout.Err(); err != nil {
			return nil, fmt.Errorf("读取 tokenizer 启动信号时出错: %w", err)
		}
		return nil, fmt.Errorf("无法从 tokenizer 服务读取启动信号")
	}
	log.Printf("Tokenizer 服务完全就绪！总启动耗时: %s", time.Since(tokenizerLoadStart))

	qdrantStart := time.Now()
	svc.qdrantConn, err = grpc.NewClient(qdrantAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("无法连接到 Qdrant: %w", err)
	}
	svc.vectorDBClient = pb.NewPointsClient(svc.qdrantConn)
	svc.collectionsClient = pb.NewCollectionsClient(svc.qdrantConn)
	_, err = svc.collectionsClient.Get(context.Background(), &pb.GetCollectionInfoRequest{CollectionName: vectorCollectionName})
	if err != nil {
		_, err = svc.collectionsClient.Create(context.Background(), &pb.CreateCollection{
			CollectionName: vectorCollectionName,
			VectorsConfig: &pb.VectorsConfig{
				Config: &pb.VectorsConfig_Params{
					Params: &pb.VectorParams{
						Size:     vectorSize,
						Distance: pb.Distance_Cosine,
					},
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("无法创建 Qdrant 集合: %w", err)
		}
		log.Printf("Qdrant 集合 '%s' 已创建，维度为 %d", vectorCollectionName, vectorSize)
	} else {
		log.Printf("已连接到 Qdrant 集合 '%s'", vectorCollectionName)
	}
	log.Printf("Qdrant 客户端初始化成功！耗时: %s", time.Since(qdrantStart))

	log.Println("--- VecX SDK 所有服务已就绪 ---")
	return svc, nil
}

// Close 函数无变化
func (s *serviceImpl) Close() {
	log.Println("--- 正在关闭 VecX SDK 服务 ---")
	if s.tokenizerService != nil && s.tokenizerService.stdin != nil {
		s.tokenizerService.stdin.Close()
	}
	if s.tokenizerService != nil && s.tokenizerService.cmd != nil && s.tokenizerService.cmd.Process != nil {
		s.tokenizerService.cmd.Process.Kill()
	}
	if s.sess != nil {
		s.sess.Destroy()
	}
	if s.qdrantConn != nil {
		s.qdrantConn.Close()
	}
	ort.DestroyEnvironment()
	log.Println("--- VecX SDK 服务已关闭 ---")
}

// Encode, AddVector, SearchSimilarVectors 函数是新的公开方法
func (s *serviceImpl) Encode(text string) ([]float32, error) {
	tokenizeStart := time.Now()
	tokens, err := s.runTokenizer(text)
	if err != nil {
		return nil, err
	}
	tokenizeDuration := time.Since(tokenizeStart)
	inferenceStart := time.Now()
	inputIDs := tokens.InputIDs[0]
	attentionMask := tokens.AttentionMask[0]
	seqLen := int64(len(inputIDs))
	inputShape := ort.NewShape(1, seqLen)
	inputIDsTensor, _ := ort.NewTensor(inputShape, inputIDs)
	defer inputIDsTensor.Destroy()
	attentionMaskTensor, _ := ort.NewTensor(inputShape, attentionMask)
	defer attentionMaskTensor.Destroy()
	inputs := []ort.Value{inputIDsTensor, attentionMaskTensor}
	outputs := make([]ort.Value, 1)
	if err := s.sess.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("ONNX 模型运行失败: %w", err)
	}
	defer outputs[0].Destroy()
	outputTensor, _ := outputs[0].(*ort.Tensor[float32])
	lastHiddenState := outputTensor.GetData()
	finalVector, _ := meanPoolingAndNormalize(lastHiddenState, attentionMask)
	inferenceDuration := time.Since(inferenceStart)
	log.Printf("VecX -> 计时分解 -> 分词通信: %s, 推理+池化: %s", tokenizeDuration, inferenceDuration)
	return finalVector, nil
}

func (s *serviceImpl) AddVector(ctx context.Context, id string, vector []float32, payload map[string]interface{}) error {
	qdrantPayload, err := mapToPayload(payload)
	if err != nil {
		return fmt.Errorf("转换 payload 失败: %w", err)
	}
	wait := true
	_, err = s.vectorDBClient.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: vectorCollectionName,
		Wait:           &wait,
		Points: []*pb.PointStruct{{
			Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: id}},
			Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: vector}}},
			Payload: qdrantPayload,
		}},
	})
	if err != nil {
		return fmt.Errorf("无法将向量存入 Qdrant: %w", err)
	}
	return nil
}

func (s *serviceImpl) SearchSimilarVectors(ctx context.Context, queryVector []float32, topK uint64) ([]*pb.ScoredPoint, error) {
	searchResult, err := s.vectorDBClient.Search(ctx, &pb.SearchPoints{
		CollectionName: vectorCollectionName,
		Vector:         queryVector,
		Limit:          topK,
		WithPayload: &pb.WithPayloadSelector{
			SelectorOptions: &pb.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("在 Qdrant 中搜索失败: %w", err)
	}
	return searchResult.GetResult(), nil
}

// 内部帮助函数 runTokenizer, meanPoolingAndNormalize 无变化
func (s *serviceImpl) runTokenizer(text string) (*TokenizerResponse, error) {
	if _, err := fmt.Fprintln(s.tokenizerService.stdin, text); err != nil {
		return nil, err
	}
	if s.tokenizerService.stdout.Scan() {
		var response TokenizerResponse
		if err := json.Unmarshal(s.tokenizerService.stdout.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("解析 tokenizer 响应失败: %w", err)
		}
		return &response, nil
	}
	return nil, fmt.Errorf("无法从 tokenizer 服务读取数据")
}
func meanPoolingAndNormalize(lastHiddenState []float32, attentionMask []int64) ([]float32, error) {
	sequenceLength := len(attentionMask)
	hiddenDim := len(lastHiddenState) / sequenceLength
	if hiddenDim != int(vectorSize) {
		return nil, fmt.Errorf("预期隐藏层维度为 %d，但得到 %d", vectorSize, hiddenDim)
	}
	pooled := make([]float32, hiddenDim)
	tokenCount := 0
	for i := 0; i < sequenceLength; i++ {
		if attentionMask[i] == 1 {
			tokenCount++
			offset := i * hiddenDim
			tokenVector := lastHiddenState[offset : offset+hiddenDim]
			for j := 0; j < hiddenDim; j++ {
				pooled[j] += tokenVector[j]
			}
		}
	}
	if tokenCount > 0 {
		for j := 0; j < hiddenDim; j++ {
			pooled[j] /= float32(tokenCount)
		}
	}
	var norm float64
	for j := 0; j < hiddenDim; j++ {
		norm += float64(pooled[j] * pooled[j])
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for j := 0; j < hiddenDim; j++ {
			pooled[j] /= float32(norm)
		}
	}
	return pooled, nil
}

// mapToPayload 是一个内部帮助函数
func mapToPayload(data map[string]interface{}) (map[string]*pb.Value, error) {
	payload := make(map[string]*pb.Value)
	for k, v := range data {
		switch val := v.(type) {
		case string:
			payload[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: val}}
		case int64:
			payload[k] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: val}}
		case int:
			payload[k] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: int64(val)}}
		case float64:
			payload[k] = &pb.Value{Kind: &pb.Value_DoubleValue{DoubleValue: val}}
		case bool:
			payload[k] = &pb.Value{Kind: &pb.Value_BoolValue{BoolValue: val}}
		default:
			return nil, fmt.Errorf("不支持的 payload 类型: %T", v)
		}
	}
	return payload, nil
}

// (在 serviceImpl 的方法区域)

// DeleteVectors 从 Qdrant 中删除一个或多个向量
func (s *serviceImpl) DeleteVectors(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// 将字符串 ID 转换为 Qdrant 需要的 PointId 格式
	var pointIds []*pb.PointId
	for _, id := range ids {
		pointIds = append(pointIds, &pb.PointId{
			PointIdOptions: &pb.PointId_Uuid{Uuid: id},
		})
	}

	wait := true
	_, err := s.vectorDBClient.Delete(ctx, &pb.DeletePoints{
		CollectionName: vectorCollectionName,
		Wait:           &wait,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: pointIds,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("从 Qdrant 删除向量失败: %w", err)
	}
	log.Printf("VecX: 成功从 Qdrant 删除了 %d 个向量", len(ids))
	return nil
}
