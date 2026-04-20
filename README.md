# Notification Service

A clean-architecture Go backend for generating and managing notifications using customizable text templates.

## Architecture

This service is structured using standard Go **Clean Architecture** patterns:
- **Models (`/models`)**: Contains core domain entities and shared data representations.
- **Repository (`/repository`)**: Contains MongoDB database logic. Exposes data through interfaces.
- **Services (`/services`)**: Contains the business logic. Acts as a bridge between the repository, handlers, and external providers.
- **Providers (`/providers`)**: Contains external API integrations for broadcasting notifications (e.g., SMTP Email, Twilio SMS).
- **Handlers (`/handlers`)**: The HTTP delivery layer. Decodes JSON requests, calls the service layer, and writes JSON responses.
- **Template (`/template`)**: A utility package specifically for parsing template strings and extracting variables.

## Tech Stack
- **Go** (Golang)
- **MongoDB** (go.mongodb.org/mongo-driver)
- **Swagger** (swaggo/http-swagger) for API documentation.

## Running Locally

1. **Configure Environment Variables:**
   We strictly use a `.env` file to prevent pushing credentials into git. Create a `.env` file in the root folder with the following structure:
   ```env

   # SMTP Email
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USER=your-email@gmail.com
   SMTP_PASSWORD=your_16_digit_app_password

   # Twilio SMS
   TWILIO_ACCOUNT_SID=your_twilio_sid
   TWILIO_AUTH_TOKEN=your_twilio_auth_token
   TWILIO_PHONE_NUMBER=+1234567890
   ```

2. **Run the server:**
   ```bash
   go run main.go
   ```
   The service will automatically load your `.env` variables and start on `http://localhost:8080`.

## API Endpoints

Once running, navigate to the Swagger UI to see active interactive documentation:
* **[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)**

### Overview of core routes:

* `GET /health` - Health check.
* `POST /templates` - Create a template (`{{variable}}` syntax supported).
* `GET /templates` - List templates.
* `GET /templates/{id}` - Get template details.
* `PUT /templates/{id}` - Update a template.
* `DELETE /templates/{id}` - Delete a template.
* `POST /notifications` - Generate a new notification for a user via a template.
* `GET /notifications/user/{user_id}` - Fetch user notifications.
* `PATCH /notifications/{id}/read` - Mark a notification as read.
