FROM ubuntu

EXPOSE 32001

ADD tacos-api /bin/tacos-api

CMD "/bin/tacos-api"
