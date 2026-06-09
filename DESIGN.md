# System Design Documentation

## Database Design
The database is designed with a security-first approach, using PostgreSQL as the primary store. The schema is optimized for student verification and auditability.

*   **Users & Profiles**: Core authentication data is stored in the `users` table (email, role, active status), while student-specific details are kept in the `profiles` table (roll number, year, branch). This separation keeps the authentication system lean.
*   **Councils**: The `councils` table defines different organizations (e.g., Science and Technology Council) that can verify student records.
*   **Verification Requests**: The `verification_requests` table tracks the lifecycle of a verification request, including the title, description, proof link, POR date, and status (`PENDING`, `APPROVED`, `REJECTED`).
*   **Scopes**: The `user_council_scopes` table maps `COUNCIL_ADMIN` users to specific councils they are authorized to manage.

## Domain Admin Verification
Security is enforced by ensuring a `COUNCIL_ADMIN` can only view and process requests for their assigned council.

*   **Scoped Access**: Each `COUNCIL_ADMIN` is assigned one or more council codes in the `user_council_scopes` table.
*   **Session Injection**: Upon authentication, the active council codes for the user are fetched from the database and injected into the request context.
*   **Middleware/Service Enforcement**: The `VerificationService` checks the `council_id` of the request against the admin's authorized scopes (`requestInReviewerScope` function) before allowing any approval or rejection.

## Student Verified Card & PDF Report
Verified records are compiled into a digital card and a downloadable PDF report.

*   **Card Generation**: The `GetVerifiedCard` service aggregates all approved verification requests for a user, grouped by council. It combines this with a snapshot of the student's profile (name, roll number, etc.) to create a structured data object.
*   **PDF Report**: The structured data is passed to `RenderReportToPDF`. This utility function serializes the data and wraps it in a minimal PDF stream (`%PDF-1.4`) that can be served directly to the browser for downloading or viewing. This ensures a consistent format without the overhead of a full rendering engine.
