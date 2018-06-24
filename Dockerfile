FROM scratch
COPY sync .
ENTRYPOINT [ "/sync" ]
