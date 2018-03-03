FROM robotalks/toolchain:latest
RUN git clone https://github.com/robotalks/mqhub.go /tmp/mqhub.go && \
    cd /tmp/mqhub.go && git checkout 3c92e551de142e623edc065a2b137cd27f01fea9 && \
    mkdir -p $GOPATH/src/github.com/robotalks/mqhub.go && \
    mv /tmp/mqhub.go/mqhub $GOPATH/src/github.com/robotalks/mqhub.go/ && \
    rm -fr /tmp/mqhub.go
