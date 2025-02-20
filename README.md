# Linky TIC exporter

Prometheus Linky TIC exporter 

## Métriques exposées

Disponible via /metrics sur le port 9100 

- linky_tic_base (gauge) - Index option Base en Wh
- linky_tic_iinst (gauge) - Intensité Instantanée en A
- linky_tic_papp (gauge) - Puissance apparente en VA

## Docker run

Execution de l'image docker

```
sudo docker run -d --env-file ./env.list -p 9100:9100 --name=linky_exporter linkytic-exporter:latest
```

## Docker build

Contruction de l'image docker

```
sudo docker build . --file Dockerfile -t linkytic-exporter
```

## Docker compose

```yaml
services:
  linkytic_exporter:
    image: dmachard/linkytic-exporter:v0.2.0
    ports:
      - "9100:9100/tcp"
    devices:
      - "/dev/ttyACM0:/dev/ttyACM0"
    environment:
      - LINKY_DEVICE=/dev/ttyACM0
      - LINKY_MODE=HISTORICAL
    restart: unless-stopped
```