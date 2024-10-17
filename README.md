# Step-by-Step Guide to Test the Current Implementation

## 1. Prerequisites:
   - Ensure you have Go installed (version 1.16 or later)
   - PostgreSQL installed and running
   - Make sure you have all the project dependencies installed

## 2. Set up the database:
   - Create a new PostgreSQL database for testing
   - Update the database connection string in the Makefile (line 2)

## 3. Run database migrations:
   - Open a terminal and navigate to the project root directory
   - Run the command: `make migrate`

## 4. Generate Swagger documentation:
   - In the same terminal, run: `make swagger`

## 5. Build the application:
   - Run the command: `make build`

## 6. Start the application:
   - Run the command: `make run`
   - The application should start, and you should see log messages indicating the server is running and the API documentation URL

## 7. Access the Swagger UI:
   - Open a web browser and go to the URL printed in the console (typically `http://localhost:8080/swagger/index.html`)
   - You should see the Swagger UI with all the available API endpoints

## 8. Test the API endpoints:

### a. Create a new server:
   - Use the `POST /servers` endpoint
   - Provide the required information in the request body (`name`, `path`, `jarFileId`, `additionalFileIds`)
   - Send the request and check the response

### b. List all servers:
   - Use the `GET /servers` endpoint
   - Send the request and verify that the created server is in the list

### c. Get a specific server:
   - Use the `GET /servers/{name}` endpoint
   - Replace `{name}` with the name of the server you created
   - Send the request and check the response

### d. Start a server:
   - Use the `POST /servers/{name}/start` endpoint
   - Replace `{name}` with the name of the server you want to start
   - Send the request and check the response

### e. Stop a server:
   - Use the `POST /servers/{name}/stop` endpoint
   - Replace `{name}` with the name of the server you want to stop
   - Send the request and check the response

### f. Send a command to a server:
   - Use the `POST /servers/{name}/command` endpoint
   - Replace `{name}` with the name of the server
   - Provide the command in the request body
   - Send the request and check the response

### g. Upload a JAR file:
   - Use the `POST /jar-files` endpoint
   - Provide the required information (`name`, `version`, `file`)
   - Send the request and check the response

### h. Upload an additional file:
   - Use the `POST /additional-files` endpoint
   - Provide the required information (`name`, `type`, `file`)
   - Send the request and check the response

### i. Delete a server:
   - Use the `DELETE /servers/{name}` endpoint
   - Replace `{name}` with the name of the server you want to delete
   - Send the request and check the response

## 9. Run unit tests:
   - In the terminal, run: `make test`
   - Check the output to ensure all tests pass

## 10. Clean up:
   - Stop the application (Ctrl+C in the terminal where it's running)
   - Run: `make clean` to remove built binaries

## 11. Additional testing:
   - Test error scenarios by providing invalid inputs or trying to perform actions on non-existent servers
   - Test concurrent requests to ensure thread safety
   - Verify that file uploads are stored correctly in the specified directories

> **Note:** Make sure to replace any placeholder values (e.g., server names, file paths) with actual values relevant to your testing environment.
