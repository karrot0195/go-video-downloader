.PHONY: run-domi run-kai

run-domi:
	go run . -watermark="Vitamin Tích Cực" -mode telegram
run-kai:
	go run . -watermark="Chuyện Cổ Hương" -mode telegram