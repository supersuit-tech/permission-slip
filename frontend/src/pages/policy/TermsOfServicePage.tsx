import { Link } from "react-router-dom";
import { PolicyLayout } from "./PolicyLayout";

export function TermsOfServicePage() {
  return (
    <PolicyLayout title="Terms of Service" lastUpdated="February 28, 2026">
      <p>
        These Terms of Service (&quot;Terms&quot;) govern your access to and use
        of Permission Slip (&quot;the Service&quot;), operated by Rocktown
        Capital LLC, doing business as Supersuit Technologies (&quot;we,&quot;
        &quot;us,&quot; or &quot;our&quot;), located in the Commonwealth of
        Virginia, USA.
      </p>
      <p>
        By creating an account or using the Service, you agree to be bound by
        these Terms and our{" "}
        <Link to="/policy/privacy">Privacy Policy</Link>. If you do not agree,
        do not use the Service.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>1. Eligibility</h2>
      <p>
        You must be at least <strong>13 years old</strong> and have the legal
        capacity to enter into a binding agreement to use the Service. By
        creating an account, you represent that you meet these requirements.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>2. Account Registration</h2>
      <p>
        Accounts are created using email-based, passwordless authentication. You
        are responsible for maintaining access to the email address associated
        with your account. Each person may maintain only one account.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>3. Service Description</h2>
      <p>
        Permission Slip is an AI agent authorization platform. It acts as an
        approval proxy between your AI agents and external third-party services,
        allowing you to review, approve, deny, or pre-authorize actions that
        agents request on your behalf.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>4. Beta Disclaimer</h2>
      <p>
        The Service is currently in <strong>beta</strong>. During the beta
        period:
      </p>
      <ul>
        <li>
          The Service is provided <strong>&quot;as is&quot;</strong> and{" "}
          <strong>&quot;as available&quot;</strong> without warranties of any kind
        </li>
        <li>Features may change, be modified, or be removed without notice</li>
        <li>
          We make no guarantees regarding uptime, availability, or reliability
        </li>
        <li>
          The beta period will be clearly communicated when it ends
        </li>
      </ul>

      {/* ------------------------------------------------------------------ */}
      <h2>5. Pricing and Payment</h2>

      <h3>5.1 What Counts as a Request</h3>
      <p>
        A &quot;request&quot; is a single action submitted by an agent for
        approval or execution. This includes:
      </p>
      <ul>
        <li>
          One-off approval requests (agent submits, you approve, action executes)
        </li>
        <li>
          Standing approval executions (pre-authorized actions that fire
          automatically)
        </li>
        <li>Denied or expired requests</li>
      </ul>
      <p>The following do <strong>not</strong> count as requests:</p>
      <ul>
        <li>
          Dashboard page views, agent registration, credential management,
          viewing audit logs, or managing standing approvals
        </li>
      </ul>

      <h3>5.2 Free Tier &mdash; $0/month</h3>
      <ul>
        <li>1,000 requests per month</li>
        <li>3 agents</li>
        <li>All built-in connectors</li>
        <li>5 active standing approvals</li>
        <li>5 stored credentials</li>
        <li>7-day audit log retention</li>
        <li>Email and web push notifications</li>
        <li>Community support (GitHub Issues)</li>
      </ul>

      <h3>5.3 Pay-as-You-Go &mdash; $0.005/request</h3>
      <ul>
        <li>First 1,000 requests per month free</li>
        <li>
          Unlimited agents, connectors, standing approvals, and credentials
        </li>
        <li>90-day audit log retention</li>
        <li>Email and web push notifications</li>
        <li>SMS available as add-on</li>
        <li>Email support</li>
      </ul>

      <h3>5.4 SMS Add-on (Paid Tier Only)</h3>
      <ul>
        <li>US/Canada: $0.01 per SMS</li>
        <li>UK/EU: $0.04 per SMS</li>
        <li>Other international: $0.05&ndash;$0.10 per SMS</li>
        <li>Pass-through carrier costs with no markup</li>
      </ul>

      <h3>5.5 Payment Terms</h3>
      <ul>
        <li>Payments are processed via Stripe</li>
        <li>Usage is invoiced monthly based on actual consumption</li>
        <li>
          <strong>All charges are final &mdash; no refunds</strong>
        </li>
        <li>Add a payment method to unlock paid tier limits</li>
        <li>SMS usage is billed as a separate line item</li>
      </ul>

      {/* ------------------------------------------------------------------ */}
      <h2>6. Acceptable Use Policy</h2>
      <p>You agree <strong>not</strong> to:</p>
      <ul>
        <li>Use the Service for illegal activities</li>
        <li>Attempt to circumvent rate limits or security controls</li>
        <li>Share account credentials</li>
        <li>
          Use agents to perform unauthorized actions on third-party services
        </li>
        <li>
          Reverse engineer the Service, except as permitted by the Apache 2.0
          open-source license
        </li>
        <li>
          Abuse the free tier (e.g., creating multiple accounts to avoid limits)
        </li>
        <li>
          Transmit malware, viruses, or harmful code through the Service
        </li>
        <li>
          Interfere with or disrupt the Service or other users&apos; use of the
          Service
        </li>
      </ul>

      {/* ------------------------------------------------------------------ */}
      <h2>7. Permission Slip as a Conduit</h2>
      <p>
        <strong>
          Permission Slip acts solely as an intermediary and conduit between you,
          your AI agents, and third-party services.
        </strong>
      </p>
      <ul>
        <li>
          We do not control, endorse, or take responsibility for the behavior of
          any AI agent, the content of any action, or the response of any
          third-party service.
        </li>
        <li>
          We facilitate the approval workflow &mdash; we do not make decisions
          about what actions should or should not be approved.
        </li>
        <li>
          We temporarily decrypt your credentials only at the moment of
          execution, solely to perform the action you approved, and do not retain
          plaintext credentials beyond that moment.
        </li>
      </ul>

      {/* ------------------------------------------------------------------ */}
      <h2>8. Open-Source Code Disclaimer</h2>
      <p>
        Permission Slip&apos;s source code is publicly available under the
        Apache 2.0 license at{" "}
        <a
          href="https://github.com/supersuit-tech/permission-slip-web"
          target="_blank"
          rel="noopener noreferrer"
        >
          github.com/supersuit-tech/permission-slip-web
        </a>
        .
      </p>
      <ul>
        <li>
          You have the ability and responsibility to review the source code
          before using the Service.
        </li>
        <li>
          By using the Service, you acknowledge that you have had the opportunity
          to inspect, audit, and evaluate the code.
        </li>
        <li>
          We are not liable for any undesirable outcomes &mdash; whether caused
          by software bugs, errors, unintended behavior, or otherwise &mdash;
          because the code is open source and available for you to review and
          verify before use.
        </li>
        <li>
          This applies to all functionality, including but not limited to:
          credential handling, action execution, approval workflows, notification
          delivery, and data storage.
        </li>
        <li>
          The Apache 2.0 license itself includes a &quot;no warranty&quot;
          clause (Section 7: &quot;no warranty of any kind, express or
          implied&quot;).
        </li>
      </ul>

      {/* ------------------------------------------------------------------ */}
      <h2>9. Credential Responsibilities</h2>
      <p>You are solely responsible for:</p>
      <ul>
        <li>
          All API keys, tokens, and credentials you store in the vault
        </li>
        <li>
          Ensuring you have proper authorization from the relevant third-party
          service to use each stored credential
        </li>
        <li>
          Ensuring credentials have appropriate permission scopes (principle of
          least privilege)
        </li>
        <li>Rotating or revoking compromised credentials immediately</li>
      </ul>
      <p>
        We encrypt credentials at rest using AES-256-GCM via Supabase Vault. We
        are not liable for unauthorized use of credentials that you validly
        stored. If a credential is used to perform an action that violates a
        third-party service&apos;s terms, that is your responsibility.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>10. Approval and Standing Approval Liability</h2>

      <h3>10.1 One-Off Approvals</h3>
      <p>
        When you approve an agent&apos;s action request, you assume full
        responsibility for the consequences of that approved action. We present
        the action details for your review &mdash; the decision to approve is
        yours alone. We are not liable for any damages, losses, or consequences
        arising from actions you chose to approve.
      </p>

      <h3>10.2 Standing Approvals</h3>
      <p>
        When you create a standing approval, you are pre-authorizing actions to
        execute automatically without per-request review. You accept full
        responsibility for all actions auto-executed under your standing
        approvals, including:
      </p>
      <ul>
        <li>
          Setting appropriate constraints (time limits, execution caps, parameter
          restrictions)
        </li>
        <li>
          Regularly reviewing and revoking standing approvals you no longer need
        </li>
      </ul>
      <p>
        We are not liable for any consequences of standing approval executions,
        even if the auto-executed action produces unexpected or undesired
        results.
      </p>

      <h3>10.3 Agent Behavior</h3>
      <p>
        We do not control or monitor what actions agents request &mdash; we only
        facilitate the approval and execution flow. You are responsible for the
        agents you register and the level of trust you grant them. If an agent
        submits a malicious or erroneous action request and you approve it,
        liability rests with you.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>11. Intellectual Property</h2>
      <p>
        The Permission Slip source code is licensed under the Apache 2.0
        license. The &quot;Permission Slip&quot; and &quot;Supersuit
        Technologies&quot; names and logos are trademarks of Rocktown Capital
        LLC. Your data remains yours &mdash; we claim no ownership of the
        content you process through the Service.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>12. Limitation of Liability</h2>
      <p>
        To the fullest extent permitted by applicable law:
      </p>
      <ul>
        <li>
          The Service is provided <strong>&quot;AS IS&quot;</strong> and{" "}
          <strong>&quot;AS AVAILABLE&quot;</strong> during beta, without warranty
          of any kind, whether express, implied, or statutory
        </li>
        <li>
          We make no warranty of uptime, availability, or fitness for a
          particular purpose
        </li>
        <li>
          <strong>
            We are not liable for actions taken by agents on your behalf,
            whether approved individually or via standing approvals
          </strong>
        </li>
        <li>
          <strong>
            We are not liable for consequences of actions executed on
            third-party services using your credentials
          </strong>
        </li>
        <li>
          <strong>
            We are not liable for the behavior, accuracy, or reliability of any
            AI agent that interacts with the Service
          </strong>
        </li>
        <li>
          We are not liable for third-party service outages, API changes, or
          third-party service responses to executed actions
        </li>
        <li>
          We are not liable for any indirect, incidental, special,
          consequential, or punitive damages
        </li>
        <li>
          We are not liable for data loss at third-party services caused by
          executed actions
        </li>
        <li>
          <strong>
            We are not liable for any outcomes caused by software defects, bugs,
            or errors
          </strong>{" "}
          &mdash; the source code is open source (Apache 2.0) and available for
          your review; by using the Service, you accept the software as-is
        </li>
      </ul>
      <p>
        <strong>
          Our total aggregate liability is capped at the fees you paid us in the
          12 months preceding the claim, or $100, whichever is greater.
        </strong>
      </p>
      <p>
        Some jurisdictions do not allow the exclusion of certain warranties or
        limitation of certain damages. In those jurisdictions, the above
        limitations apply to the fullest extent permitted by law.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>13. Indemnification</h2>
      <p>
        You agree to indemnify and hold harmless Rocktown Capital LLC (d/b/a
        Supersuit Technologies), its officers, directors, employees, and agents
        from any claims, damages, losses, or expenses (including reasonable
        attorney fees) arising from:
      </p>
      <ul>
        <li>
          <strong>Actions executed using your credentials</strong> &mdash; any
          third-party claims resulting from actions performed on external
          services
        </li>
        <li>
          <strong>Approved agent actions</strong> &mdash; consequences of actions
          you chose to approve, whether one-off or standing
        </li>
        <li>
          <strong>Credential misuse</strong> &mdash; storing unauthorized,
          stolen, or improperly scoped credentials
        </li>
        <li>
          <strong>Third-party terms violations</strong> &mdash; actions that
          violate the terms of service of external services accessed through
          Permission Slip
        </li>
        <li>
          <strong>Your agents</strong> &mdash; any claims arising from the
          behavior of agents you registered and authorized
        </li>
        <li>
          <strong>Software defects</strong> &mdash; any losses arising from
          bugs, errors, or unintended behavior in the open-source code that you
          had the opportunity to review
        </li>
        <li>
          <strong>Misuse of the Service</strong> &mdash; violation of these Terms
          or any applicable law
        </li>
      </ul>

      {/* ------------------------------------------------------------------ */}
      <h2>14. Dispute Resolution</h2>

      <h3>14.1 Informal Resolution</h3>
      <p>
        Before filing any formal dispute, you agree to attempt to resolve it
        informally by contacting us at{" "}
        <a href="mailto:support@supersuit.tech">support@supersuit.tech</a>. We
        will attempt to resolve the dispute within <strong>30 days</strong> of
        receiving your notice.
      </p>

      <h3>14.2 Binding Arbitration</h3>
      <p>
        If informal resolution fails, all disputes will be resolved by{" "}
        <strong>binding arbitration</strong> under the American Arbitration
        Association (AAA) Commercial Arbitration Rules. Arbitration will be
        conducted in Virginia. The arbitrator&apos;s decision is final and
        binding.
      </p>

      <h3>14.3 Class Action Waiver</h3>
      <p>
        <strong>
          All claims must be brought in your individual capacity, not as a
          plaintiff or class member in any purported class, collective, or
          representative proceeding.
        </strong>{" "}
        You agree to waive any right to participate in a class action.
      </p>

      <h3>14.4 Small Claims Exception</h3>
      <p>
        Either party may bring claims in small claims court if the claim
        qualifies.
      </p>

      <h3>14.5 Costs</h3>
      <p>
        Each party bears its own arbitration costs. AAA fees are split per AAA
        rules.
      </p>

      <h3>14.6 Opt-Out</h3>
      <p>
        You may opt out of the arbitration agreement within{" "}
        <strong>30 days of creating your account</strong> by emailing{" "}
        <a href="mailto:support@supersuit.tech">support@supersuit.tech</a> with
        the subject line &quot;Arbitration Opt-Out.&quot;
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>15. Termination</h2>
      <ul>
        <li>
          You may delete your account at any time via Settings.
        </li>
        <li>
          We may suspend or terminate accounts for violations of these Terms with
          reasonable notice, except for egregious violations, which may result in
          immediate termination.
        </li>
        <li>
          Upon termination, all data is deleted within 30 days, except as
          required by law or for legitimate business purposes (e.g., payment
          records).
        </li>
        <li>Accrued payment obligations survive termination.</li>
        <li>
          Sections that by their nature should survive termination will survive,
          including limitation of liability, indemnification, and dispute
          resolution.
        </li>
      </ul>

      {/* ------------------------------------------------------------------ */}
      <h2>16. Changes to These Terms</h2>
      <p>
        We may update these Terms from time to time. When we make material
        changes, we will notify you by email at least{" "}
        <strong>30 days in advance</strong> and update the &quot;Last
        updated&quot; date at the top of this page. Your continued use of the
        Service after the 30-day notice period constitutes acceptance of the
        updated Terms.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>17. Severability</h2>
      <p>
        If any provision of these Terms is found to be unenforceable or invalid
        by a court or arbitrator, the remaining provisions will continue in full
        force and effect.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>18. Entire Agreement</h2>
      <p>
        These Terms, together with our{" "}
        <Link to="/policy/privacy">Privacy Policy</Link> and{" "}
        <Link to="/policy/cookies">Cookie Policy</Link>, constitute the entire
        agreement between you and Rocktown Capital LLC regarding your use of the
        Service.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>19. Governing Law</h2>
      <p>
        These Terms are governed by the laws of the{" "}
        <strong>Commonwealth of Virginia, USA</strong>, without regard to
        conflict of law principles.
      </p>

      {/* ------------------------------------------------------------------ */}
      <h2>20. Contact Us</h2>
      <p>
        If you have questions about these Terms, contact us at:
      </p>
      <ul>
        <li>
          <strong>Email:</strong>{" "}
          <a href="mailto:support@supersuit.tech">support@supersuit.tech</a>
        </li>
        <li>
          <strong>Entity:</strong> Rocktown Capital LLC (d/b/a Supersuit
          Technologies)
        </li>
        <li>
          <strong>Governing law:</strong> Commonwealth of Virginia, USA
        </li>
      </ul>
    </PolicyLayout>
  );
}
