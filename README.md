# Loadbalancer

First attempt at writing a basic loadbalancer. Includes:
- TCP and HTTP proxying
- Target health checks that automatically remove/adds targets to the available pool
- Prometheus metrics
- Dockerfile

To run the Docker image and attach (must have built locally first):
`docker run -it --rm -p 8000:8000 -p 8001:8001 -p 8091:8091 -v $(pwd)/config.yaml:/opt/loadbalancer/config.yaml loadbalancer`
