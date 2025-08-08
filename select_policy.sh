# Change directory to the script's location
cd $(dirname $0)
go run ./... -policy $1
