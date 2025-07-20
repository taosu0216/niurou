import sys
import json
from transformers import AutoTokenizer

print("Python: Tokenizer loading...", file=sys.stderr)

# --- 最终修复：现在脚本的工作目录就是 SDK 根目录，所以可以直接引用 ---
tokenizer = AutoTokenizer.from_pretrained('mpnet_onnx')

print("TOKENIZER_READY")
sys.stdout.flush() 

print("Python: Tokenizer is ready and waiting for input.", file=sys.stderr)

while True:
    try:
        text = sys.stdin.readline().strip()
        if not text:
            break

        encoded_input = tokenizer(
            text,
            padding=True,
            truncation=True,
            return_tensors='np'
        )

        response = {
            "input_ids": encoded_input['input_ids'].tolist(),
            "attention_mask": encoded_input['attention_mask'].tolist(),
        }

        print(json.dumps(response))
        sys.stdout.flush()

    except Exception as e:
        print(f"Tokenizer error: {e}", file=sys.stderr)
        break