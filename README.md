# go-test-task
**how to use**

- To start, please install: docker, docker-compose and go

 - `'make up'` on root folder run it will start test-task docker
   container with all requied dependencies. 
  - `'make bash'` to open container bash window 
  - `localhost:3333` to access go container from localhost
  - `make aws-cli foo bar` to execute the aws cli with parameters foo bar
  - `make aws-cli sqs list-queues` to access the sqs queue on aws localstack
  - If you import dependencies to your go code, please use `make stop` and `make up` again to automatically download them
  - To access the sql server, use mysql:3306 as adress and Port.
  - In your go application you have to use the eu-central-1 region to access aws localstack


To Run:

Run the server with go run cmd/hello-world/main.go
Test the api:
```sh
curl --location --request GET 'http://localhost:8000/v1/' --header 'test: key'
```
Run the all test: go test ./...