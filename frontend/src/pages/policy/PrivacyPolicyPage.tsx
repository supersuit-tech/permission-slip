import { Link } from "react-router-dom";
import { PolicyLayout } from "./PolicyLayout";

export function PrivacyPolicyPage() {
  return (
    <PolicyLayout title="Privacy Policy" lastUpdated="February 28, 2026">
      <p>
        Supersuit Technologies LLC
        (&quot;we,&quot; &quot;us,&quot; or &quot;our&quot;), operates Permission
        Slip (&quot;the Service&quot;) at{" "}
        <a
          href="https://app.permissionslip.dev"
          target="_blank"
          rel="noopener noreferrer"
        >
          app.permissionslip.dev
        </a>
        . This Privacy Policy explains what personal information we collect, how
        we use it, who we share it with, and your rights regarding that
        information.
      </p>
      <p>
        By creating an account or using the Service, you agree to the collection
        and use of information as described in this policy. If you do not agree,
        please do not use the Service.
      </p>

      <h2>1. Information We Collect</h2>

      <h3>1.1 Information You Provide at Signup</h3>
      <ul>
        <li>
          <strong>Email address</strong> &mdash; used for passwordless OTP
          authentication via Supabase
        </li>
        <li>
          <strong>Username</strong> &mdash; 3&ndash;32 characters, chosen during
          onboarding
        </li>
      </ul>

      <h3>1.2 Optional Profile Information</h3>
      <ul>
        <li>
          <strong>Contact email</strong> &mdash; for email notifications (may
          differ from your login email)
        </li>
        <li>
          <strong>Phone number</strong> &mdash; in E.164 format, for SMS
          notifications
        </li>
      </ul>

      <h3>1.3 Automatically Collected Information</h3>
      <ul>
        <li>
          <strong>User ID</strong> &mdash; a UUID assigned by Supabase
          authentication
        </li>
        <li>
          <strong>Profile creation timestamp</strong>
        </li>
        <li>
          <strong>Session data</strong> &mdash; Supabase authentication session
          tokens (JWTs)
        </li>
      </ul>

      <h3>1.4 Sensitive Information You Store</h3>
      <ul>
        <li>
          <strong>API keys and credentials</strong> for third-party services
          &mdash; encrypted at rest with AES-256-GCM via Supabase Vault.
          Permission Slip never has access to plaintext credentials outside of
          the moment of action execution.
        </li>
      </ul>

      <h3>1.5 Agent and Operational Data</h3>
      <ul>
        <li>Agent registrations (name, public key, metadata)</li>
        <li>
          Approval requests (action type, parameters, status, timestamps)
        </li>
        <li>Standing approval configurations</li>
        <li>
          Audit events (approvals, denials, cancellations, executions)
        </li>
      </ul>

      <h2>2. How We Use Your Information</h2>
      <p>We use the information we collect to:</p>
      <ul>
        <li>Authenticate you and manage your sessions</li>
        <li>
          Process agent approval requests and execute authorized actions on your
          behalf
        </li>
        <li>
          Send notifications (email, SMS, web push) based on your preferences
        </li>
        <li>Maintain audit trails for security and accountability</li>
        <li>
          Analyze usage patterns to improve the Service (operational metadata
          only &mdash; see Section 4)
        </li>
        <li>Detect abuse and enforce rate limits</li>
      </ul>

      <h2>3. Third-Party Services</h2>
      <p>
        We share information with the following third-party service providers as
        necessary to operate the Service:
      </p>
      <table>
        <thead>
          <tr>
            <th>Service</th>
            <th>Purpose</th>
            <th>Data Shared</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Supabase</td>
            <td>Authentication, database, credential vault</td>
            <td>Email, user ID, session tokens, all application data</td>
          </tr>
          <tr>
            <td>Stripe</td>
            <td>Payment processing</td>
            <td>Email, payment method, billing address</td>
          </tr>
          <tr>
            <td>SendGrid (optional)</td>
            <td>Email notifications</td>
            <td>Contact email, notification content</td>
          </tr>
          <tr>
            <td>Twilio (optional)</td>
            <td>SMS notifications</td>
            <td>Phone number, notification content</td>
          </tr>
          <tr>
            <td>Google Fonts</td>
            <td>Font loading via CSS</td>
            <td>IP address (standard browser request)</td>
          </tr>
          <tr>
            <td>Web Push (VAPID)</td>
            <td>Browser push notifications</td>
            <td>Push subscription endpoint</td>
          </tr>
        </tbody>
      </table>

      <h2>4. Data Sharing Categories</h2>
      <p>
        We draw a strict distinction between two categories of data when it
        comes to analytics, marketing, or any future third-party integrations:
      </p>

      <h3>4.1 Operational Metadata (May Be Shared)</h3>
      <p>
        The following data may be shared with analytics or marketing platforms to
        help us understand product usage and improve the Service:
      </p>
      <ul>
        <li>Email address and user ID</li>
        <li>Aggregate usage counts (requests per period, active agents)</li>
        <li>
          Connector names and action types used (e.g., &quot;gmail,&quot;
          &quot;email.send&quot;)
        </li>
        <li>Agent IDs and agent names</li>
        <li>Timestamps, event types, and outcomes (approved, denied, etc.)</li>
        <li>
          Feature usage patterns (pages visited, settings configured)
        </li>
      </ul>

      <h3>4.2 Request Content (Never Shared)</h3>
      <p>
        The following data is <strong>never</strong> shared with any third party,
        including analytics or marketing platforms:
      </p>
      <ul>
        <li>
          Action parameters (e.g., email body, recipient, subject line)
        </li>
        <li>Approval context and confirmation codes</li>
        <li>Stored credentials and API keys</li>
        <li>Standing approval constraint details</li>
        <li>
          Any data passed through Permission Slip to or from third-party
          services on your behalf
        </li>
      </ul>

      <h2>5. Data Retention</h2>
      <table>
        <thead>
          <tr>
            <th>Data Type</th>
            <th>Retention Period</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Audit logs (Free tier)</td>
            <td>7 days</td>
          </tr>
          <tr>
            <td>Audit logs (Paid tier)</td>
            <td>90 days</td>
          </tr>
          <tr>
            <td>Account data (profile, credentials, agents)</td>
            <td>Until you delete your account</td>
          </tr>
          <tr>
            <td>Payment records</td>
            <td>As required by law (typically 7 years)</td>
          </tr>
          <tr>
            <td>Notification logs</td>
            <td>Transient (not permanently stored)</td>
          </tr>
        </tbody>
      </table>

      <h2>6. Your Rights</h2>
      <p>
        Depending on your jurisdiction, you may have some or all of the
        following rights with respect to your personal information:
      </p>
      <ul>
        <li>
          <strong>Access</strong> &mdash; View all your data through the
          Permission Slip dashboard.
        </li>
        <li>
          <strong>Deletion</strong> &mdash; Delete your account and all
          associated data via Settings.
        </li>
        <li>
          <strong>Portability</strong> &mdash; Export your audit logs from the
          dashboard.
        </li>
        <li>
          <strong>Correction</strong> &mdash; Update your profile information in
          Settings.
        </li>
        <li>
          <strong>Notification opt-out</strong> &mdash; Disable any notification
          channel independently in your notification preferences.
        </li>
      </ul>
      <p>
        To exercise any of these rights beyond what is available in the
        dashboard, contact us at{" "}
        <a href="mailto:support@supersuit.tech">support@supersuit.tech</a>.
      </p>

      <h2>7. Data Hosting</h2>
      <p>
        All data is stored and processed in the <strong>United States</strong>.
        If you are accessing the Service from outside the United States, please
        be aware that your information will be transferred to, stored, and
        processed in the United States.
      </p>

      <h2>8. Children&apos;s Privacy</h2>
      <p>
        Permission Slip is not directed at children. You must be at least{" "}
        <strong>13 years old</strong> to create an account. We do not knowingly
        collect personal information from children under 13. If we become aware
        that we have collected data from a child under 13, we will delete it
        promptly. If you believe a child under 13 has provided us with personal
        information, please contact us at{" "}
        <a href="mailto:support@supersuit.tech">support@supersuit.tech</a>.
      </p>

      <h2>9. Security</h2>
      <p>
        We implement the following security measures to protect your
        information:
      </p>
      <ul>
        <li>
          Credentials encrypted at rest using AES-256-GCM via Supabase Vault
        </li>
        <li>
          Passwordless authentication &mdash; no passwords are stored
        </li>
        <li>Ed25519 cryptographic agent identity verification</li>
        <li>JWT-based session management</li>
        <li>HTTPS enforced for all connections</li>
        <li>Per-request approval with unique confirmation codes</li>
        <li>
          Comprehensive security headers (Content Security Policy, HSTS, and
          others)
        </li>
      </ul>
      <p>
        While we take commercially reasonable measures to protect your data, no
        method of transmission over the Internet or electronic storage is 100%
        secure. We cannot guarantee absolute security.
      </p>

      <h2>10. Cookies and Local Storage</h2>
      <p>
        For detailed information about the cookies and local storage mechanisms
        we use, please see our{" "}
        <Link to="/policy/cookies">Cookie Policy</Link>.
      </p>

      <h2>11. Changes to This Policy</h2>
      <p>
        We may update this Privacy Policy from time to time. When we make
        material changes, we will notify you by email and update the &quot;Last
        updated&quot; date at the top of this page. Your continued use of the
        Service after notification constitutes acceptance of the updated policy.
      </p>

      <h2>12. Contact Us</h2>
      <p>
        If you have questions about this Privacy Policy or our data practices,
        contact us at:
      </p>
      <ul>
        <li>
          <strong>Email:</strong>{" "}
          <a href="mailto:support@supersuit.tech">support@supersuit.tech</a>
        </li>
        <li>
          <strong>Entity:</strong> Supersuit Technologies LLC
        </li>
        <li>
          <strong>Governing law:</strong> Commonwealth of Virginia, USA
        </li>
      </ul>
    </PolicyLayout>
  );
}
