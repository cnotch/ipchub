FROM alpine:latest
RUN mkdir /app
WORKDIR /app
COPY bin/docker .
# Expose ipchub ports
EXPOSE 554
CMD ["./ipchub"]