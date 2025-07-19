 docker run -p 6333:6333 -p 6334:6334 \                  1 main?
    -v $(pwd)/qdrant_storage:/qdrant/storage \
    qdrant/qdrant

docker run -d \
    -p 7474:7474 -p 7687:7687 \
    -v $(pwd)/neo4j_data:/data \
    -e NEO4J_AUTH=neo4j/password \
    --name neo4j-agent-db \
    neo4j:latest