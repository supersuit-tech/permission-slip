import { Link } from "react-router-dom";
import { PolicyLayout } from "./PolicyLayout";

export function CookiePolicyPage() {
  return (
    <PolicyLayout title="Cookie Policy" lastUpdated="February 28, 2026">
      <p>
        This Cookie Policy explains how Rocktown Capital LLC, doing business as
        Supersuit Technologies (&quot;we,&quot; &quot;us,&quot; or
        &quot;our&quot;), uses cookies and similar local storage technologies
        when you use Permission Slip at{" "}
        <a
          href="https://app.permissionslip.dev"
          target="_blank"
          rel="noopener noreferrer"
        >
          app.permissionslip.dev
        </a>
        .
      </p>

      <h2>1. What Are Cookies and Local Storage?</h2>
      <p>
        <strong>Cookies</strong> are small text files placed on your device by a
        website. They are widely used to make websites work efficiently and to
        provide information to site owners.
      </p>
      <p>
        <strong>Local storage</strong> and <strong>session storage</strong> are
        browser-based mechanisms that allow websites to store data on your device.
        Unlike cookies, this data is not sent to the server with every request.
        Session storage is cleared when you close your browser tab; local storage
        persists until explicitly cleared.
      </p>

      <h2>2. How We Use Cookies and Local Storage</h2>
      <p>
        Permission Slip does not use advertising pixels, behavioral tracking,
        or marketing tools. We use a single, privacy-focused analytics service
        (PostHog) that is <strong>only activated after you accept cookies</strong>{" "}
        via our consent banner.
      </p>
      <p>
        All storage we use falls into the following categories:
      </p>

      <h3>2.1 Essential (Required)</h3>
      <p>
        These are necessary for the Service to function and cannot be disabled.
      </p>
      <table>
        <thead>
          <tr>
            <th>Storage Mechanism</th>
            <th>Key / Name</th>
            <th>Purpose</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Local storage</td>
            <td>Supabase auth session tokens</td>
            <td>Authentication and session management via the Supabase client</td>
          </tr>
          <tr>
            <td>Cookie</td>
            <td>
              <code>ps_consent</code>
            </td>
            <td>Remembering your cookie consent choice (cross-subdomain)</td>
          </tr>
          <tr>
            <td>Session storage</td>
            <td>
              <code>mfa_pending_enrollment</code>
            </td>
            <td>Temporary state during multi-factor authentication setup</td>
          </tr>
        </tbody>
      </table>

      <h3>2.2 Functional</h3>
      <p>
        These enhance your experience but are not strictly required for the
        Service to operate.
      </p>
      <table>
        <thead>
          <tr>
            <th>Storage Mechanism</th>
            <th>Key / Name</th>
            <th>Purpose</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Local storage</td>
            <td>
              <code>permission-slip-theme</code>
            </td>
            <td>Remembering your dark/light mode selection</td>
          </tr>
        </tbody>
      </table>

      <h3>2.3 Analytics (Consent Required)</h3>
      <p>
        We use <strong>PostHog</strong>, a privacy-focused product analytics
        platform, to understand how people use Permission Slip (e.g., feature
        adoption, navigation patterns). PostHog analytics is{" "}
        <strong>only active when you accept cookies</strong> via our consent
        banner. If you decline or have not yet responded, no analytics data is
        collected or sent.
      </p>
      <table>
        <thead>
          <tr>
            <th>Storage Mechanism</th>
            <th>Key / Name</th>
            <th>Purpose</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>In-memory only</td>
            <td>PostHog anonymous ID</td>
            <td>
              Temporary identifier for the current session; not persisted to
              disk and cleared on page reload
            </td>
          </tr>
        </tbody>
      </table>
      <p>
        PostHog analytics data is limited to operational metadata only (see
        our <Link to="/policy/privacy">Privacy Policy</Link>, Section 4, for
        the distinction between operational metadata and request content). No
        emails, names, or other PII are included in analytics events. Your IP
        address is not sent to PostHog.
      </p>

      <h2>3. Managing Your Preferences</h2>

      <h3>3.1 Our Cookie Consent Banner</h3>
      <p>
        When you first visit Permission Slip, a cookie consent banner allows you
        to accept or reject non-essential cookies. You can change your preference
        at any time by clearing your browser&apos;s cookies for our site,
        which will cause the banner to reappear on your next visit.
      </p>

      <h3>3.2 Browser Settings</h3>
      <p>
        Most browsers allow you to control cookies and local storage through
        their settings. You can typically:
      </p>
      <ul>
        <li>View and delete existing cookies and stored data</li>
        <li>Block all or certain types of cookies</li>
        <li>Set preferences for specific websites</li>
      </ul>
      <p>
        Please note that blocking essential storage may prevent the Service from
        functioning correctly. Specifically, blocking local storage or cookies
        for our site will prevent you from logging in or remembering your
        consent preferences.
      </p>

      <h2>4. Changes to This Policy</h2>
      <p>
        We will update this Cookie Policy when we add new categories of cookies
        or change how we use existing ones. When material changes are made:
      </p>
      <ul>
        <li>We will update the &quot;Last updated&quot; date at the top of this page</li>
        <li>If we introduce a new cookie category (e.g., analytics), we will re-prompt your consent via the cookie consent banner</li>
        <li>We will notify you of material changes via email</li>
      </ul>

      <h2>5. Contact Us</h2>
      <p>
        If you have questions about our use of cookies or local storage, contact
        us at{" "}
        <a href="mailto:support@supersuit.tech">support@supersuit.tech</a>.
      </p>
      <p>
        For broader information about how we handle your data, please see our{" "}
        <Link to="/policy/privacy">Privacy Policy</Link>.
      </p>
    </PolicyLayout>
  );
}
