module github.com/GeoloeG-IsT/gollem

go 1.18

replace github.com/GeoloeG-IsT/gollem => /home/ubuntu/gollem_github

replace github.com/GeoloeG-IsT/gollem/pkg/core => /home/ubuntu/gollem_github/pkg/core

replace github.com/GeoloeG-IsT/gollem/pkg/providers/openai => /home/ubuntu/gollem_github/pkg/providers/openai

require github.com/joho/godotenv v1.5.1 // indirect
