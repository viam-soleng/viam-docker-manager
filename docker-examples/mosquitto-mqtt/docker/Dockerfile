# syntax=docker/dockerfile:1

FROM eclipse-mosquitto:latest
ADD mosquitto-no-auth.conf /mosquitto-no-auth.conf
ENTRYPOINT ["mosquitto", "-c", "/mosquitto-no-auth.conf"]
