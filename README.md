# LinkHub

Bảng liên kết (bookmark) cá nhân chạy trong mạng LAN. Toàn bộ giao diện được nhúng vào binary Go, không cần cài đặt runtime hay cơ sở dữ liệu bên ngoài.

## Tính năng

- Thêm, sửa, xóa liên kết; nhấp vào thẻ để mở trang trong tab mới.
- Đánh dấu yêu thích (lọc nhanh), gán **nhiều danh mục** cho một liên kết.
- Chọn danh mục có sẵn trong biểu mẫu hoặc tạo danh mục mới ngay tại chỗ; quản lý danh mục (thêm, đổi tên, xóa) ở sidebar.
- Tìm kiếm tức thời, không phân biệt dấu tiếng Việt (`mang` khớp `mạng`), khớp cả tên danh mục.
- Sắp xếp thủ công (kéo thả), mới nhất, xem nhiều, A → Z.
- Đếm lượt truy cập và thời điểm truy cập gần nhất.
- Tự động lấy tiêu đề và mô tả từ URL khi thêm liên kết.
- Kiểm tra trạng thái các liên kết (HEAD, tự chuyển sang GET khi cần), hiển thị chấm xanh/đỏ trên biểu tượng.
- Ba mức hiển thị thẻ: **Mặc định**, **Gọn**, **Nhỏ**; lựa chọn được ghi nhớ.
- Giao diện sáng/tối, ghi nhớ theo trình duyệt.
- Xuất/nhập dữ liệu JSON để sao lưu hoặc chuyển máy.
- Responsive cho desktop, tablet và điện thoại.

### Phím tắt

| Phím | Chức năng |
| --- | --- |
| `/` hoặc `Ctrl/Cmd + K` | Tập trung ô tìm kiếm |
| `N` | Thêm liên kết mới |
| `Esc` | Đóng menu/hộp thoại hoặc xóa nội dung tìm kiếm |

## Yêu cầu

- Go 1.27 trở lên.
- Không dùng thư viện ngoài; chỉ cần Go standard library.

## Cài đặt và chạy

Chạy trực tiếp từ mã nguồn:

```bash
go run ./cmd/linkhub
```

Hoặc build rồi chạy:

```bash
go build -trimpath -ldflags="-s -w" -o bin/linkhub.exe ./cmd/linkhub
./bin/linkhub.exe
```

Trên Linux/macOS:

```bash
go build -trimpath -ldflags="-s -w" -o bin/linkhub ./cmd/linkhub
./bin/linkhub
```

Mặc định ứng dụng lắng nghe cổng `8080` và lưu dữ liệu trong thư mục `data/`. Khi khởi động, log sẽ in địa chỉ truy cập:

```text
LinkHub đang chạy tại http://localhost:8080
Trong mạng LAN: http://192.168.0.10:8080
```

### Tham số dòng lệnh

| Tham số | Mặc định | Mô tả |
| --- | --- | --- |
| `-addr` | `:8080` | Địa chỉ lắng nghe, ví dụ `:8080`, `127.0.0.1:8080` hoặc `192.168.0.10:8080` |
| `-data` | `data` | Thư mục chứa file dữ liệu `links.json` |

Ví dụ chỉ cho phép truy cập từ chính máy:

```bash
./bin/linkhub.exe -addr 127.0.0.1:8080
```

## Dữ liệu và sao lưu

- Dữ liệu nằm tại `data/links.json`, được tạo tự động cùng 5 liên kết mẫu trong lần chạy đầu tiên.
- Mỗi lần ghi, ứng dụng ghi file tạm rồi đổi tên để tránh mất dữ liệu khi ghi dở.
- Định dạng phiên bản 2: danh mục được lưu riêng trong `categories`, mỗi liên kết tham chiếu bằng `categoryIds`. File phiên bản 1 (trường `category` dạng chuỗi) được tự động chuyển đổi khi khởi động và khi nhập dữ liệu.
- Xóa danh mục chỉ gỡ danh mục khỏi các liên kết, không xóa liên kết.
- Sao lưu: dùng nút tải xuống trong sidebar hoặc gọi `GET /api/export`.
- Khôi phục: dùng nút tải lên trong sidebar (chế độ **merge**) hoặc gọi `POST /api/import?mode=replace` để thay thế toàn bộ.

Ví dụ dữ liệu (rút gọn):

```json
{
  "version": 2,
  "categories": [
    { "id": "a1b2c3", "name": "Phát triển", "order": 0, "createdAt": "2026-11-12T02:00:00Z" }
  ],
  "links": [
    {
      "id": "f00dfeed",
      "title": "MDN Web Docs",
      "url": "https://developer.mozilla.org",
      "categoryIds": ["a1b2c3"],
      "order": 0,
      "createdAt": "2026-11-12T02:00:00Z"
    }
  ]
}
```

## API

| Phương thức | Đường dẫn | Mô tả |
| --- | --- | --- |
| `GET` | `/api/links` | Danh sách liên kết đã sắp xếp |
| `POST` | `/api/links` | Tạo liên kết mới |
| `PUT` | `/api/links/{id}` | Cập nhật liên kết |
| `DELETE` | `/api/links/{id}` | Xóa liên kết |
| `POST` | `/api/links/{id}/visit` | Ghi nhận một lượt truy cập |
| `POST` | `/api/reorder` | Đổi thứ tự, body `{"ids": ["id1", "id2"]}` |
| `GET` | `/api/categories` | Danh sách danh mục |
| `POST` | `/api/categories` | Tạo danh mục, body `{"name": "Tên"}`; trả `200` kèm danh mục đã có nếu trùng tên (không phân biệt hoa thường), `201` nếu tạo mới |
| `PUT` | `/api/categories/{id}` | Đổi tên danh mục, body `{"name": "Tên mới"}`; `409` khi tên đã tồn tại |
| `DELETE` | `/api/categories/{id}` | Xóa danh mục và gỡ nó khỏi mọi liên kết |
| `GET` | `/api/export` | Tải file JSON sao lưu |
| `POST` | `/api/import` | Nhập JSON; thêm `?mode=replace` để thay thế toàn bộ |
| `GET` | `/api/metadata?url=...` | Lấy tiêu đề/mô tả của trang |
| `POST` | `/api/check` | Kiểm tra trạng thái, body `{"urls": ["..."]}` (tối đa 100 URL) |

Ví dụ:

```bash
# Tạo liên kết với một hoặc nhiều danh mục (có thể bỏ qua https://)
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"url": "example.com", "title": "Example", "categoryIds": ["<id-danh-muc>"]}'

# Tạo danh mục mới
curl -X POST http://localhost:8080/api/categories \
  -H "Content-Type: application/json" \
  -d '{"name": "Tiện ích"}'

# Sao lưu
curl -o linkhub-backup.json http://localhost:8080/api/export

# Khôi phục, thay thế toàn bộ dữ liệu hiện có
curl -X POST "http://localhost:8080/api/import?mode=replace" \
  -H "Content-Type: application/json" \
  --data-binary @linkhub-backup.json
```

Chỉ URL `http` và `https` được chấp nhận; URL thiếu scheme sẽ tự động được thêm `https://`. Trường `category` (chuỗi) kiểu cũ vẫn được chấp nhận khi tạo/cập nhật/nhập dữ liệu và sẽ tự động ánh xạ sang danh mục tương ứng.

## Cấu trúc thư mục

```text
Link-Dashboard/
├── cmd/linkhub/main.go          # Entrypoint: cờ, khởi động, graceful shutdown
├── internal/
│   ├── link/link.go             # Model Link và Category dùng chung
│   ├── store/store.go           # Lưu trữ JSON, ghi nguyên tử
│   ├── server/
│   │   ├── server.go            # Server, routes, phục vụ SPA
│   │   ├── handlers.go          # CRUD liên kết/danh mục, visit, reorder, import/export
│   │   ├── metadata.go          # Lấy tiêu đề/mô tả từ URL
│   │   ├── probe.go             # Kiểm tra trạng thái liên kết
│   │   ├── helpers.go           # Chuẩn hóa URL, JSON helpers
│   │   └── middleware.go        # Logging, panic recovery
│   └── webui/
│       ├── embed.go             # go:embed giao diện vào binary
│       ├── index.html
│       └── static/
│           ├── app.css
│           ├── app.js
│           └── favicon.svg
├── bin/                         # Kết quả build (không commit)
├── data/                        # Dữ liệu chạy (không commit)
├── go.mod
└── .gitignore
```

## Phát triển

```bash
gofmt -w cmd internal
go vet ./...
go build -trimpath -ldflags="-s -w" -o bin/linkhub.exe ./cmd/linkhub
```

Lưu ý: giao diện được nhúng bằng `go:embed`, vì vậy sau khi sửa `internal/webui` (HTML/CSS/JS) cần build lại binary thì thay đổi mới có hiệu lực.

## Bảo mật

Ứng dụng **không có đăng nhập hay phân quyền**, phù hợp cho mạng LAN tin cậy như bookmark cá nhân. Với giá trị mặc định `-addr :8080`, server lắng nghe trên mọi giao diện mạng của máy.

- Không mở cổng này ra Internet hoặc qua router NAT.
- Nếu chỉ dùng trên máy hiện tại, chạy với `-addr 127.0.0.1:8080`.
- Nếu dùng trong LAN, giới hạn truy cập bằng firewall của hệ điều hành khi cần.
