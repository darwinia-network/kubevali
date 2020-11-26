FROM alpine

COPY bin/kubevali_linux_amd64 /kubevali
ENTRYPOINT [ "/kubevali" ]
