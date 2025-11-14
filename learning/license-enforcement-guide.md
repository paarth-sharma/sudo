# License Enforcement Guide

## How to Enforce Your Commons Clause + MIT License

Your software is now protected under the MIT License with Commons Clause. Here's how to monitor and enforce it.

---

## 1. Monitoring for Violations

### Automated Monitoring

**GitHub Search Alerts:**
- Set up Google Alerts for: `"SUDO Kanban" OR "sudo kanban board" site:*.com`
- Monitor GitHub forks and stars on your repository
- Use GitHub's "Used by" feature to see who's using your code

**Web Monitoring:**
- Google search periodically: `"powered by SUDO" OR "using SUDO Kanban"`
- Search for your unique code patterns or function names
- Monitor cloud marketplaces (AWS Marketplace, Azure, etc.)

**Tools You Can Use:**
- **Google Alerts** - Free, automated email notifications
- **Mention.com** - Social media and web monitoring (paid)
- **GitHub Search API** - Programmatic monitoring of forks

### Manual Checks

Check these platforms quarterly:
- Product Hunt
- Indie Hackers
- Reddit (r/SaaS, r/golang, r/selfhosted)
- Hacker News
- Cloud provider marketplaces

---

## 2. Identifying Violations

### Clear Violations ❌

Someone is violating your license if they:

1. **Sell the software:**
   - Charging money to download or use SUDO
   - Selling licenses to use SUDO
   - Bundling it in a paid software package

2. **Offer as a paid service:**
   - "SUDO as a Service" with monthly fees
   - Paid hosted instances
   - Freemium model with paid tiers

3. **Charge for hosting/support:**
   - "We'll host SUDO for you for $X/month"
   - Paid managed hosting services
   - Consulting fees where SUDO is the primary value

### Gray Areas (Usually violations)

- Hosting SUDO and charging for "other services" where SUDO is the main value
- Free tier + paid features built on top of SUDO
- Using SUDO internally but charging clients for access

### Allowed Uses ✅

These are perfectly fine:
- Self-hosting for internal company use (even if the company is for-profit)
- Modifying and using in a larger free/open-source project
- Using SUDO to manage tasks for clients (where you charge for your services, not SUDO)
- Running SUDO for your team/organization without charging anyone
- Consulting where you help others self-host (if incidental to broader services)

---

## 3. Enforcement Process

### Step 1: Document the Violation

Before taking action, gather evidence:

**Screenshot/Archive:**
- Pricing pages showing fees
- Service descriptions
- Terms of service
- About pages
- Marketing materials

**Use Archive.org:**
```
https://web.archive.org/save/[their-url]
```

**Save locally:**
- Full page screenshots
- HTML source code
- Dates and timestamps
- URLs

### Step 2: Initial Contact (Friendly)

**Email Template:**

```
Subject: SUDO Kanban License Inquiry

Hi [Name/Team],

I'm Paarth Sharma, creator of SUDO Kanban. I noticed you're using SUDO
in your project/service at [URL].

SUDO is licensed under MIT with Commons Clause, which means it's free
for personal use and self-hosting, but commercial services require a
separate license agreement.

From what I can see, it appears you may be [describe what they're doing -
e.g., "offering paid hosted instances"]. If this is correct, you would
need a commercial license to continue operating.

I'd love to work with you on this. We have a few options:

1. Transition to a free self-hosted model (fully compliant with the license)
2. Obtain a commercial license (we can discuss terms)
3. Remove SUDO from your service

Could we schedule a brief call to discuss? I'm happy to work out an
arrangement that benefits both of us.

Best regards,
Paarth Sharma
[Your contact info]
[Link to LICENSE file]
```

**Why start friendly:**
- Many violations are unintentional
- They might not have read the license
- You could gain a commercial customer
- Better PR for your project
- Faster resolution

### Step 3: Formal Cease & Desist (If Needed)

If they ignore your friendly email after 2 weeks, send a formal notice:

**Formal Notice Template:**

```
Subject: Formal Notice: SUDO Kanban License Violation

[Their Company Name]
[Date]

RE: Unauthorized Commercial Use of SUDO Kanban Software

Dear [Name],

This letter serves as formal notice that [Company Name] is in violation
of the software license for SUDO Kanban ("the Software").

FACTS:
- SUDO Kanban is licensed under MIT License with Commons Clause
- The Commons Clause explicitly prohibits selling the software or
  offering it as a service for compensation
- Your service at [URL] appears to [describe violation]
- I contacted you on [date] but received no response

REQUIRED ACTION:
You must immediately:
1. Cease all commercial use of SUDO Kanban, OR
2. Obtain a valid commercial license within 14 days

CONSEQUENCES OF NON-COMPLIANCE:
- Copyright infringement claim under applicable law
- Pursuit of damages and legal fees
- Public disclosure of the violation

I remain open to resolving this amicably through a licensing agreement.
Please respond within 7 days to discuss.

Sincerely,
Paarth Sharma
Creator, SUDO Kanban
[Contact Information]

Attachments:
- LICENSE file
- Evidence of violation (screenshots, archives)
```

**Important:** Consider having a lawyer review this letter before sending.

### Step 4: Legal Action (Last Resort)

If they still don't comply:

**Options:**

1. **DMCA Takedown (for websites):**
   - File with their hosting provider
   - File with their cloud provider (AWS, Azure, etc.)
   - Effective and fast for many cases

2. **Lawyer Letter:**
   - Hire an IP attorney to send a formal demand
   - Cost: $500-2,000
   - Often effective immediately

3. **Lawsuit:**
   - Only for serious/profitable violations
   - Copyright infringement damages can be substantial
   - Expensive ($10,000+) but you can recover legal fees

**When to consider legal action:**
- They're making significant money ($10k+ revenue)
- They ignored multiple notices
- They're damaging your reputation
- You can afford it or find a lawyer who works on contingency

---

## 4. Preventing Violations

### Add License Notices in the Code

Add this to your main.go:

```go
const LicenseNotice = `
SUDO Kanban - Copyright (c) 2025 Paarth Sharma
Licensed under MIT License with Commons Clause

This software is free for personal use and self-hosting.
Commercial use, selling, or offering as a service requires
a separate commercial license.

Contact: [your-email] for commercial licensing.
`

func main() {
    log.Println(LicenseNotice)
    // ... rest of your code
}
```

### Add to HTML Footer

In your templates, add:

```html
<footer class="text-xs text-gray-500 text-center py-4">
    SUDO Kanban © 2025 Paarth Sharma |
    <a href="/license">License</a> |
    Free for personal use |
    <a href="mailto:your-email">Commercial Licensing</a>
</footer>
```

### Create a /license Route

Add an endpoint that displays your license clearly:

```go
protected.GET("/license", func(c *gin.Context) {
    c.Header("Content-Type", "text/plain")
    c.String(200, readLicenseFile())
})
```

### Watermark in UI (Optional)

Add a subtle "Licensed under Commons Clause" in the app:
- In settings page
- In about section
- In help documentation

---

## 5. Monetization Strategies (While Enforcing)

### Dual Licensing

Offer two licenses:
1. **Free (Commons Clause):** Personal use, self-hosting
2. **Commercial:** $X/month or $Y one-time for commercial use

### Examples of Commercial License Terms:

**Small Business:**
- $49/month or $499/year
- Up to 50 users
- Email support

**Enterprise:**
- $299/month or $2,999/year
- Unlimited users
- Priority support
- Custom features

### Create a License Purchase Page

Add to your repository:
- `COMMERCIAL-LICENSE.md` explaining pricing
- Payment link (Stripe, Gumroad, etc.)
- Contact form for custom deals

---

## 6. Building Goodwill

### Be Reasonable

**Good enforcement:**
- Give warnings before taking action
- Offer affordable commercial licenses
- Support community self-hosting
- Praise proper attribution

**Bad enforcement:**
- Immediate lawsuits
- Unreasonable pricing
- Going after individuals
- Aggressive tactics

### Public Communication

If you do enforce, be professional:

**Good tweet:**
> "Reminder: SUDO Kanban is free for self-hosting but requires a commercial
> license for paid services. If you're building on SUDO, let's talk!
> I want to support your success while protecting the project."

**Bad tweet:**
> "Just caught another company STEALING my code! Lawyers are involved!"

---

## 7. Real-World Examples

### Success Stories

**Redis Labs:**
- Switched to source-available license in 2018
- Sent friendly emails to violators
- Converted many to commercial customers
- Grew revenue significantly

**Elastic:**
- Moved to Elastic License 2.0
- Enforced against AWS (high-profile case)
- Protected their business model
- Community largely supported them

### Key Lessons

1. **Most violations are accidental** - people don't read licenses
2. **Friendly first works** - many will comply or buy a license
3. **Document everything** - you'll need it later
4. **Be consistent** - enforce against everyone equally
5. **Offer fair commercial terms** - don't be greedy

---

## 8. Resources

### Legal Resources

- [Electronic Frontier Foundation (EFF)](https://www.eff.org/) - Digital rights help
- [Software Freedom Law Center](https://softwarefreedom.org/) - Free legal advice for FOSS
- [Creative Commons](https://creativecommons.org/) - License information

### Monitoring Tools

- **Google Alerts:** alerts.google.com
- **GitHub Search:** github.com/search
- **Archive.org:** web.archive.org (save evidence)

### Commercial License Templates

- [Indie Hackers Commercial License Examples](https://www.indiehackers.com/)
- [Fair Source License](https://fair.io/) - Alternative licensing model
- [FOSSA Commons Clause Guide](https://github.com/fossas/commons-clause)

---

## 9. Quick Reference Checklist

**When you spot a potential violation:**

- [ ] Screenshot/archive the evidence
- [ ] Verify it's actually a violation (reread license)
- [ ] Send friendly email with options
- [ ] Wait 14 days for response
- [ ] Send formal notice if no response
- [ ] Wait 7-14 days
- [ ] Consider DMCA takedown or lawyer letter
- [ ] Last resort: Legal action

**Remember:**
- Most people will comply when asked nicely
- A commercial customer is better than a lawsuit
- Your goal is protecting your work, not punishing people
- Document everything from day one

---

## Contact for License Violations

If you discover someone violating this license, you can:

1. Send them this document
2. Refer them to the LICENSE file
3. Direct them to contact you for commercial licensing

**Your Contact Info:**
- Email: [your-email]
- GitHub: @[your-username]
- Website: [your-site]

---

## Summary

**The Commons Clause gives you:**
- ✅ Legal protection against commercial exploitation
- ✅ Ability to monetize through dual licensing
- ✅ Control over how your work is used commercially
- ✅ Free personal/self-hosted use for the community

**To enforce it effectively:**
- 👀 Monitor regularly
- 📧 Start friendly
- 📜 Document everything
- ⚖️ Escalate only when necessary
- 💰 Offer fair commercial terms

Your license is enforceable and legally binding. Use these tools wisely to protect your work while building a sustainable project.
