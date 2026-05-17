# EC2 Deployment

This guide deploys the project to one AWS EC2 instance with Docker Compose. It is intended for a simple demo deployment, not a high-availability production system.

## Architecture

```text
Internet
  -> EC2 public IP:8080
      -> order-service
          -> product-service over Docker network
          -> payment-service over Docker network
          -> Postgres over Docker network
          -> Redis over Docker network
          -> Kafka over Docker network
              -> notification-service
```

Only `order-service` is published to the EC2 host. Postgres, Redis, Kafka, product-service, payment-service, and notification-service stay inside the Docker network.

## Recommended EC2 Instance

- AMI: Ubuntu 22.04 LTS or Ubuntu 24.04 LTS
- Instance type: `t3.small` minimum, `t3.medium` recommended
- Storage: 20 GB or more

Kafka can be memory-hungry. If containers restart or Kafka fails to start on `t3.small`, use `t3.medium`.

## Security Group

Inbound rules:

```text
22    TCP    your IP only        SSH
8080  TCP    0.0.0.0/0           order-service HTTP API
```

Do not open these ports publicly for this compose deployment:

```text
5432  Postgres
6379  Redis
9092  Kafka
50051 product-service gRPC
50052 payment-service gRPC
```

## Server Setup

SSH into the instance:

```bash
ssh -i path/to/key.pem ubuntu@EC2_PUBLIC_IP
```

Install Docker and Git:

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl git
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker ubuntu
```

Log out and SSH back in so the Docker group change takes effect.

## Deploy

Clone the repository:

```bash
git clone https://github.com/AlexW6385/Distributed-Order-Processing-System.git
cd Distributed-Order-Processing-System
```

Create the environment file:

```bash
cp .env.example .env
```

Edit `.env` and change at least:

```env
POSTGRES_PASSWORD=replace-with-a-strong-password
ORDER_SERVICE_PORT=8080
GIN_MODE=release
```

Start the system:

```bash
make prod-up
```

Check containers:

```bash
make prod-ps
```

Check health:

```bash
curl http://localhost:8080/health
curl http://EC2_PUBLIC_IP:8080/health
```

Expected response:

```json
{"database":"ok","redis":"ok","status":"ok"}
```

## Smoke Test

List products:

```bash
curl http://EC2_PUBLIC_IP:8080/products
```

Create an order:

```bash
PRODUCT_ID=$(curl -s http://localhost:8080/products | jq -r '.products[0].id')

curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: ec2-demo-001' \
  -d "{
    \"customer_email\": \"demo@example.com\",
    \"items\": [
      {
        \"product_id\": \"$PRODUCT_ID\",
        \"quantity\": 1
      }
    ]
  }"
```

Pay an order:

```bash
curl -X POST http://localhost:8080/orders/ORDER_ID/pay \
  -H 'Content-Type: application/json' \
  -H 'X-Request-ID: ec2-demo-001' \
  -d '{"idempotency_key":"ec2-payment-001"}'
```

Watch logs:

```bash
make prod-logs
```

You should see JSON logs from the HTTP request, downstream gRPC calls, outbox publishing, and notification consumption.

## Operations

Pull the latest code and redeploy:

```bash
git pull
make prod-up
```

Stop containers without deleting data:

```bash
make prod-down
```

Reset all containers and delete Postgres data:

```bash
make prod-reset
```

Use `prod-reset` only when you are comfortable deleting all local EC2 demo data.

## Notes

- This deployment keeps Postgres, Redis, and Kafka on the same EC2 instance for simplicity.
- For a more production-grade AWS setup, move Postgres to RDS, Redis to ElastiCache, Kafka to MSK, and the services to ECS Fargate.
- Store real secrets outside Git. `.env` is ignored by Git; `.env.example` is only a template.
