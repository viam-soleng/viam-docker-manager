# Mosquitto MQTT Broker Image Example

This example explains how you can use an official Docker Hub image (eclipse-mosquitto in this particular example), modify and upload it to make it deployable thrugh the Viam docker manager module.

## Create / Modify Dockerfile

```s
# syntax=docker/dockerfile:1

FROM eclipse-mosquitto:latest                               // The base image we want to modify
ADD mosquitto-no-auth.conf /mosquitto-no-auth.conf          // The modified config file we want to use
ENTRYPOINT ["mosquitto", "-c", "/mosquitto-no-auth.conf"]   // The new entrypoint / startup command of the MQTT broker
```

## Build a New Image

I use [Docker Hub](https://hub.docker.com/) but any image registry would work, to make my images accessible. Therefore you will likely have to log in first.

```shell
docker auth
```

To then build the image locally, using the previously configured Dockerfile, you can use the following command:

```shell
docker build --tag your-repo/your-image .
```

And to upload the image, use this command:

```shell
docker push your-repo/your-image:latest
```

## Viam Configuration

To then use the image as part of a machine configuration, add the following to your component attributes.
You will get the `repo_digest` when you upload the image to the docker registry.

```json
{
  "run_options": {
    "host_options": {
      "Binds": "viam:/opt",
      "NetworkMode": "default",
      "PortBindings": [
        "1883:1883"
      ],
      "AutoRemove": true
    }
  },
  "image_name": "your-repo/your-image:latest",
  "repo_digest": "sha256:4<--YOUR DIGEST-->"
}
```
