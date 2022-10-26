FROM golang:1.19 as image
MAINTAINER Patryk Orzechowski, Joost Wagenaar

#setup env variables
ENV PENNSIEVE API_HOST host
#Currently inactive, use token
#ENV PENNSIEVE_API_KEY key
ENV PENNSIEVE_API_TOKEN key
ENV PENNSIEVE_API_SECRET secret
ENV PENNSIEVE_PATH .

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
