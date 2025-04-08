# gRPC File Server

Сервис на Go, работающий по gRPC и реализующий 3 метода:

- Загрузку изображений (Upload)
- Скачивание файлов (Download)
- Просмотр списка загруженных файлов (ListFiles)
- Ограничение количества одновременных подключений

## Запуск

```bash
go run server/main.go

### Клиент

По умолчанию, клиент делает следующее:
    Загружает image.png в директорию uploaded_files/ на сервере
    Запрашивает список всех файлов ListFiles
    Скачивает файл обратно как downloaded_image.png

Ограничения по подключению
    Максимум 10 одновременных Upload/Download запросов
    Максимум 100 одновременных запросов списка файлов
При превышении — клиент получит ошибку too many concurrent ... requests.





