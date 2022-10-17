FROM golang:1.19 as image
MAINTAINER Patryk Orzechowski, Joost Wagenaar

#setup env variables
ARG PENNSIEVE_PATH
ARG API_KEY
ARG API_SECRET

WORKDIR /opt/pennsieve

#copy all files from the repo
COPY . .

#check what is copied
#RUN ls -la $PENNSIEVE_PATH/*

RUN go build -v -o /opt/pennsieve/$PENNSIEVE_PATH .
RUN echo "${{ env.PENNSIEVE_PATH }}"
RUN go run /opt/pennsieve/$PENNSIEVE_PATH/main.go config init ${{ env.PENNSIEVE_PATH }} --api_token=${{ env.PENNSIEVE_API_KEY}} --api_secret=${{ env.PENNSIEVE_API_SECRET }} -f
RUN go run /opt/pennsieve/$PENNSIEVE_PATH/main.go agent start

#Running: sudo docker build . --build-arg 'PENNSIEVE_PATH=.' --build-arg 'API_KEY=key' --build-arg 'API_SECRET=secret'