FROM golang:1.19 as image
MAINTAINER Patryk Orzechowski, Joost Wagenaar

#setup env variables
ENV PENNSIEVE API_HOST host
ENV PENNSIEVE_API_KEY key
ENV PENNSIEVE_API_SECRET secret
ENV PENNSIEVE_PATH .
ENV PENNSIEVE_DATASET none
ENV PENNSIEVE_UPLOAD_BUCKET none
ENV PENNSIEVE_AGENT_PORT 9000
ENV PENNSIEVE_AGENT_UPLOAD_WORKERS 1
ENV PENNSIEVE_AGENT_CHUNK_SIZE 32

WORKDIR /opt/pennsieve

#copy all files from the repo
COPY . .
RUN apt-get update
RUN go install 
RUN go build -v -o /opt/pennsieve .
RUN ls -laR /opt/pennsieve/
RUN echo "$PENNSIEVE_PATH"
RUN ln -s -f pennsieve-agent pennsieve

EXPOSE 9000
CMD ["go", "run", "main.go", "agent", "start"]
