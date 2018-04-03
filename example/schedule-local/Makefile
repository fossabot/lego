start:
		CONFIG_URI=file://${PWD}/config.json go run main.go

schedule:
		@curl -X "POST" "http://localhost:3000/job/foo" \
				-H 'Content-Type: application/json; charset=utf-8' \
				-d '{"In": 10, "Data": "Random data"}'
		@echo ""
