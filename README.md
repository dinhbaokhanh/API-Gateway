# PTIT Gateway - Nền Tảng Mạng Xã Hội Học Tập PTIT

Dự án xây dựng nền tảng mạng xã hội. Repository này chứa mã nguồn phần **API Gateway** cho hệ thống microservices.

## 1. Mục Tiêu Dự Án

*   **Mục đích:** Viết một API Gateway cho hệ thống microservices nhằm mục đích học tập và rèn luyện kỹ năng cũng như sử dụng cho đồ án môn học.

---

## 2. Tính Năng & Chức Năng Cốt Lõi

*   **Post & Nội dung:** Quản lý bài viết đa dạng trạng thái, Code Editor + Execution Sandbox (review diff), bộ lọc nội dung ẩn danh (Toxic detection).
*   **Quản lý Nhóm/Cộng đồng:** Task board, Lịch nhóm, chia sẻ tiến độ thiết kế Figma/Code nội bộ.
*   **Search & Đề xuất (AI tích hợp):** Graph matching algorithms (Neo4j), Semantic Search, Transformer xử lý tổng hợp tự động hỏi đáp.
*   **Quản lý người dùng & file:** Xác thực, phân quyền Dashboard kiểm duyệt chuyên sâu, Cloud storage tài liệu học tập cá nhân và tập thể.

---

## 3. Định Hướng Phát Triển API Gateway (Kiến trúc Configuration-Driven theo KrakenD)

Thay vì lập trình cứng (hard-code) các logic định tuyến và xử lý ngay trong mã nguồn gốc, API Gateway của hệ thống sẽ được tái cấu trúc và phát triển dựa trên triết lý cốt lõi của **KrakenD** (Configuration-driven API Gateway). Hướng đi này giúp tách biệt hoàn toàn phần xử lý logic (Engine) khỏi phần khai báo dịch vụ (Declaration), đồng thời hỗ trợ mạnh mẽ các mẫu kiến trúc microservices nâng cao.

Các tính năng cốt lõi sẽ tập trung xây dựng bao gồm:

### 3.1. Khởi chạy và Định tuyến bằng File Cấu Hình (Configuration-Driven Routing)
*   **Mô tả:** Thay vì code cứng (hard-code) các route như `"/api/users"` hay `"/api/orders"` vào trong bộ định tuyến (router) của Go, toàn bộ hệ thống sẽ sử dụng một file cấu hình duy nhất (ví dụ: `gateway.json` hoặc `gateway.yaml`).
*   **Hoạt động:** Khi Gateway khởi động, nó sẽ nạp (parse) file cấu hình này lên memory, tự động tạo các dynamic routes (đường dẫn động) và cấu hình các downstream services (dịch vụ đích) tương ứng.
*   **Lợi ích:** Dễ dàng thêm, sửa, xóa, hoặc quy hoạch lại API mapping mà không cần recompile mã nguồn.

### 3.2. Kiến trúc Backend for Frontend (BFF) thông qua Cấu Hình
*   **Mô tả:** Tạo ra các backend riêng biệt được may đo (tailor-made) cho từng loại frontend cụ thể (như Mobile App, Web App).
*   **Hoạt động:** Thay vì viết thêm Service trung gian, Gateway đảm nhận vai trò BFF. Cấu hình các endpoint trả về các payload khác nhau cho cùng một tài nguyên tùy thuộc vào client.

### 3.3. Cung cấp dữ liệu tập trung (Endpoint Aggregation / Data Fetching)
*   **Mô tả:** Hệ thống hỗ trợ nhận 1 request từ Client và tự động phát tách (fan-out) ra nhiều request gọi tới các Microservices.
*   **Hoạt động:** Gateway sử dụng Goroutines để gọi song song các service, trộn (merge) phản hồi lại thành một JSON object hợp nhất và trả về Client. Giảm độ trễ mạng và over-fetching.

### 3.4. Quản lý Middleware (Interceptor) tự động
*   **Mô tả:** Áp dụng hệ thống Middleware (Xác thực, CORS, Rate-Limiting, Logging) ở tầng Gateway.
*   **Hoạt động:** Tắt/bật Middleware thông qua cấu hình `gateway.json` ở cấp độ Toàn cục (Global) hoặc Cục bộ (Endpoint-level).

### 3.5. Kiến trúc Phi trạng thái (Stateless Architecture)
*   **Mô tả:** API Gateway không kết nối trực tiếp với Database hay Session Cache.
*   **Hoạt động:** Mọi thao tác ủy quyền được xác minh tính hợp lệ độc lập (verify JWT) hoặc forward về Auth Service, giúp Gateway dễ dàng scale.

---

## 4. Cấu Trúc Kỹ Thuật (Đang triển khai)
Dự án được viết bằng **Go**, hiện đang có cấu trúc như sau (sẽ được tái cấu trúc dần để đáp ứng kiến trúc mới):

### Cấu Trúc Thư Mục
```text
ptit-gateway/
|-- cmd/
|   `-- gateway/
|       `-- main.go          # Điểm khởi động
|-- internal/
|   |-- app/                 # Khởi tạo và chạy HTTP server
|   |-- config/              # Đọc biến môi trường
|   |-- routing/             # Khai báo route proxy -> backend services
|   |-- middleware/          # CORS, logger, recover, (sau này có rate-limit)
|   `-- proxy/               # Reverse proxy implementation
|-- go.mod
`-- README.md
```

### Các Biến Môi Trường (Mẫu)
- `PORT`: Cổng gateway (mặc định `8080` hoặc có thể thay đổi)
- `BACKEND_USERS`: URL service Users (Xác thực, Thông tin)
- `BACKEND_POSTS`: URL service Q&A, Thảo luận
- `BACKEND_GROUPS`: URL service Nhóm học tập, Kanban
- `BACKEND_SEARCH`: URL service Semantic Search & Analytics

### Build & Run
```bash
go mod tidy
go run ./cmd/gateway
```
