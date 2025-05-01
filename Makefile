.PHONY: format lint run # 告訴 make 這些不是檔案名字，是「行為名稱」（避免跟目錄或檔案重名衝突）

format:
	go fmt ./...
	goimports -w .

lint:
	staticcheck ./...

check: format lint

run:
	go run main.go


install-tools:
	go install tool 

# 整理並驗證依賴
tidy:
	go mod tidy
