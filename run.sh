clear

mkdir -p .build
gin --build=./cmd/graphql --bin=./.build/api run

# ---

# export FUNCTION_TARGET=GraphQL
# go run ./cmd/knative