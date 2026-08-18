docker stop partir-core-console-1
docker rm partir-core-console-1
docker rmi partir-core-console
docker image prune -f
docker compose -f docker-compose.prod.yml up -d --build console
