# Detailed System Design Document
- single golang project
- pure js (no ts) in frontend/sw


## Introduction

This document outlines the design and implementation details of a notification system that leverages a server-client architecture using SQLite, a custom input binary (`sendnotif`), an exchange directory for message handling, and a Progressive Web App (PWA) on the client side. The system aims to provide a simple yet robust mechanism for sending notifications to authenticated clients while allowing for topic-based filtering managed entirely on the client side.

## System Overview

### Architecture

- **Server**: Hosts the SQLite database, processes incoming notifications, and sends push notifications to registered clients.
- **Input Binary (`sendnotif`)**: A command-line tool used to send notifications by creating files in a designated exchange directory.
- **Exchange Directory**: A filesystem-based queue where notification files are placed for the server to process.
- **Exchange Package (`exchange`)**: A shared package that handles reading, writing, and defining notification files, used by both the server and the `sendnotif` binary.
- **Client (PWA)**: Receives notifications, handles client-side topic filtering, and manages authentication.

### Code Organization

- **Exchange Package (`exchange`)**:
  - Provides data structures and methods for notification files.
  - Ensures consistent handling of notifications between the server and `sendnotif`.
  - Encapsulates file operations and validation logic.

## Data Handling

### SQLite Database Structure

The server uses an SQLite database to store information about devices and topics. The database contains the following tables:

1. **`devices`**:
   - **Columns**:
     - `device_id` (Primary Key)
     - `public_key`
     - `auth_token`
     - `registration_date`

   - **Purpose**: Stores information about registered devices eligible to receive notifications.

2. **`topics`**:
   - **Columns**:
     - `topic_id` (Primary Key)
     - `topic_name`
     - `creation_date`

   - **Purpose**: Contains a list of all topics generated on-the-fly as notifications are received.

3. **`notifications`**:
   - **Columns**:
     - `notification_id` (Primary Key)
     - `topic_id` (Foreign Key referencing `topics`)
     - `timestamp`
     - `message`
     - `metadata` (JSON blob for any additional data)

   - **Purpose**: Stores all notifications along with their associated topics.

### Topic Management

- **Dynamic Creation**: When a notification with a new topic is received, the server adds the topic to the `topics` table if it doesn't already exist.
- **Client Retrieval**: Clients can request a list of all topics from the server to manage their local filtering preferences.

## Notification Input

### Input Binary (`sendnotif`)

The `sendnotif` binary is a command-line tool that abstracts the complexity of sending notifications. It simplifies the process by handling the creation of properly formatted notification files in the exchange directory.

#### Responsibilities:

- Parse command-line arguments:
  ```bash
  sendnotif --topic "System Updates" --priority "High" --message "Server maintenance at 10 PM."
  ```
- Validate input data to ensure it meets the required format.
- Use the `exchange` package to create a notification file in the `pending` directory.

### Exchange Directory Structure

The exchange directory is structured to facilitate smooth communication between the `sendnotif` tool and the server.

- **`/path/to/exchange/pending/`**: Holds notification files waiting to be processed.
- **`/path/to/exchange/errors/`**: Stores invalid or failed notification files for debugging purposes.

### Exchange Package (`exchange`)

The `exchange` package is a separate module that handles all notification exchange operations.

#### Features:

- **Notification Definition**: Provides data structures for notifications, including topic, metadata, and message body.
- **File Operations**: Implements reading and writing of notification files.
- **Validation**: Contains methods to validate the structure and content of notifications.
- **Error Handling**: Manages invalid files by moving them to the `errors` directory.

### Notification File Format

The notification files are plain text files inspired by email formatting for simplicity and readability.

**Example Format**:

```
System Updates
Date: 2024-11-29 14:00:00
Priority: High
-----
Server maintenance is scheduled for 10 PM tonight. Please save your work.
```

#### Structure:

1. **First Line**: Topic name (e.g., `System Updates`).
2. **Optional Metadata**: Key-value pairs for additional information (e.g., `Date`, `Priority`).
3. **Separator**: A clear marker (`-----`) to indicate the end of the header.
4. **Message Body**: The content of the notification.

### Server Processing

A background process on the server continuously monitors the `pending` directory for new notification files.

#### Steps:

1. **File Detection**: The server detects a new file in the `pending` directory.
2. **Validation**:
   - Uses the `exchange` package to parse and validate the notification file.
   - Extracts the topic, metadata, and message body.
3. **Database Insertion**:
   - Adds the topic to the `topics` table if it's new.
   - Inserts the notification into the `notifications` table.
4. **Notification Dispatch**:
   - Sends the notification to all registered devices.
5. **File Handling**:
   - On success: Deletes the file from `pending`.
   - On failure: Moves the file to `errors` and logs an error notification.

## Server Notification Logic

### Message Flow

#### Receiving Notifications:

- The server listens for new files in the `pending` directory.
- Upon detecting a file, it initiates the validation and processing sequence using the `exchange` package.

#### Saving to SQLite:

- **Topics**: Ensures the topic exists in the `topics` table.
- **Notifications**: Inserts a new record into the `notifications` table with all relevant data.

#### Pushing Notifications:

- Iterates through all entries in the `devices` table.
- Sends the notification using the Web Push Protocol.
- Implements retry logic for failed attempts.

### Error Handling

#### Invalid File Format:

- Moves the problematic file to the `errors` directory.
- Logs an error notification under a reserved topic, such as `Errors`.

#### Push Failures:

- Retries sending the notification a predefined number of times.
- If all retries fail, logs the failure and possibly deregisters the device after repeated failures.

## Client-Side (PWA) Details

### Styling with Tailwind CSS

The PWA utilizes Tailwind CSS for all styling needs, enabling rapid UI development with a utility-first approach.

- **Advantages**:
  - Consistent design without writing custom CSS.
  - Responsive design out of the box.
  - Simplifies the styling process, allowing developers to focus on functionality.

### Client Authentication

To prevent unauthorized access to notifications, the client must authenticate with the server.

#### Authentication Mechanism:

1. **Registration Request**:
   - The client initiates a registration request to the server.
   - This could be triggered manually within the PWA by the user.

2. **Server-Side Code Generation**:
   - The server generates a unique, time-limited authentication code (e.g., valid for 5 minutes).
   - This code is displayed in the server logs or via a command-line interface accessible only to authorized personnel.

3. **Client Input**:
   - The user retrieves the code from the server (requiring server access).
   - Enters the code into the PWA's authentication prompt.

4. **Verification**:
   - The client sends the code to the server for verification.
   - Upon successful verification, the server issues an `auth_token` associated with the `device_id`.

5. **Token Storage**:
   - The client securely stores the `auth_token` for future communications.

#### Security Considerations:

- **Time-Limited Codes**: Ensures that codes cannot be reused indefinitely.
- **Server Access Requirement**: Only individuals with server access can retrieve the authentication code, limiting potential unauthorized registrations.
- **Token Validation**: The server validates the `auth_token` on each request to ensure only authenticated devices receive notifications.

### Receiving Notifications

- The client registers for push notifications using the standard Web Push API.
- All notifications are received regardless of topic.
- Upon receipt, the client checks the topic against its locally stored "ignored" list.
- If the topic is not ignored, the notification is displayed using the browser's `showNotification` method.

### Managing Topics

- **Fetching Topics**:
  - The client periodically requests the list of all topics from the server.
  - This could be done at startup or when the user accesses the topic management interface.

- **Ignoring Topics**:
  - Users can mark topics as "ignored" within the PWA.
  - The ignored list is stored locally using `IndexedDB` or `localStorage`.

- **Local Filtering**:
  - Since all notifications are sent to all clients, filtering is entirely client-side, providing flexibility and reducing server complexity.

## Security Considerations

### Authentication Mechanisms

- **One-Time Codes**: The use of server-generated, time-limited codes ensures that only authorized users can register devices.
- **Token-Based Authentication**: Clients use a long-lived `auth_token` for subsequent communications, which the server validates.
- **Server Access Requirement**: Restricting code retrieval to individuals with server access adds an extra layer of security.

### Push API Security Features

- **VAPID (Voluntary Application Server Identification)**:
  - Used to authenticate the server sending the push notifications.
  - Ensures that push messages are sent by a trusted server.

- **Encryption**:
  - Payloads are encrypted using keys exchanged during the subscription process.
  - Only the intended client can decrypt the notification content.

- **Subscription Management**:
  - Clients can unsubscribe at any time.
  - The server should handle unsubscribe requests and remove devices from the `devices` table accordingly.

## Advantages of This Approach

- **Scalability**:
  - The server's responsibility is minimized by sending all notifications to all clients without managing individual preferences.
  - Client-side filtering allows for personalized experiences without additional server load.

- **Flexibility**:
  - Clients can implement complex filtering, notification handling, and user interface enhancements independently of the server.

- **Simplicity**:
  - The use of plain-text files and an exchange directory simplifies debugging and maintenance.
  - The architecture is straightforward, making it easier to containerize and deploy.

- **Reusability**:
  - The `exchange` package promotes code reuse and consistency between the server and `sendnotif` binary.

- **Security**:
  - Authentication mechanisms ensure that only authorized devices receive notifications.
  - The reliance on server access for code retrieval limits potential unauthorized registrations.

- **Robustness**:
  - Error handling mechanisms ensure that issues are logged and can be addressed promptly.
  - The system is designed to handle failures gracefully, moving erroneous files to an `errors` directory for inspection.

- **Efficient Styling**:
  - Tailwind CSS streamlines the styling process, reducing the need for custom CSS and enhancing development speed.

## Conclusion

The proposed notification system provides a balanced approach between simplicity and functionality. By leveraging an exchange directory and a custom input binary, it abstracts the complexities of inter-process communication. The introduction of the `exchange` package ensures consistent handling of notification files between the server and `sendnotif`, enhancing code maintainability and reuse. The server remains lightweight by offloading topic filtering to the client, which enhances scalability and flexibility. The authentication mechanism, while simple, effectively ensures that only authorized clients receive notifications, aligning with the security requirements. Additionally, using Tailwind CSS for styling simplifies the frontend development process and ensures a consistent user interface.
