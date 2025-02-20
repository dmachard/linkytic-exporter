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