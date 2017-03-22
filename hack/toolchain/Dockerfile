FROM robotalks/toolchain:latest
RUN mkdir -p $GOPATH/src/github.com/robotalks \
        $GOPATH/src/github.com/eclipse \
        $GOPATH/src/github.com/rs && \
    git clone https://github.com/eclipse/paho.mqtt.golang $GOPATH/src/github.com/eclipse/paho.mqtt.golang && \
    git clone https://github.com/rs/xid $GOPATH/src/github.com/rs/xid && \
    git clone https://github.com/robotalks/mqhub.go $GOPATH/src/github.com/robotalks/mqhub.go && \
    git clone https://github.com/robotalks/talk.contract $GOPATH/src/github.com/robotalks/talk.contract && \
    cd $GOPATH/src/github.com/eclipse/paho.mqtt.golang && git checkout v1.0.0 && \
    cd $GOPATH/src/github.com/rs/xid && git checkout 0c3fc84215f5aed2aa490b8b1638869cea0bd337 && \
    cd $GOPATH/src/github.com/robotalks/mqhub.go && git checkout 3c92e551de142e623edc065a2b137cd27f01fea9 && \
    cd $GOPATH/src/github.com/robotalks/talk.contract && git checkout c5f5f934790f7dcdf7d8a2c3e5b75baaeed5a69f
