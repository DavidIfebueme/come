@channel

Backend Engineers — Stage 3 Task

Build Insighta Labs+: Secure Access & Multi-Interface Integration

In this stage, you are given a Technical Requirements Document for Insighta Labs+. Your job is to take the Profile Intelligence System from Stage 2 and turn it into a platform that real users can actually log into, with roles, sessions, and two working interfaces.

Stage 2 stays intact. Filtering, sorting, pagination, natural language search - all of it. Break anything and it counts against you.

What you are building:
• GitHub OAuth with PKCE, for both the CLI and the browser
• Access + refresh token management with short expiry windows
• Role-based access control - admin and analyst, enforced across every endpoint
• API versioning and an updated pagination shape
• CSV profile export
• A globally installable CLI tool that stores credentials at ~/.insighta/credentials.json
• A web portal with HTTP-only cookies and CSRF protection
• Rate limiting and request logging

Three separate repos. One backend. Everything talks to the same system.

You'll be graded out of 100. The pass mark is 75.

How you will be evaluated:
System design qualitySecurity implementationConsistency across componentsHandling of edge casesCode quality
What to submit:
• Backend repo
• CLI repo
• Web portal repo
• Live backend URL
• Live web portal URL

Your README must cover:
System architectureAuthentication flowCLI usageToken handling approachRole enforcement logicNatural language parsing approach

TECHNICAL REQUIREMENTS DOCUMENT - link
AIRTABLE LINK - link

SUBMISSION - Run /submit in stage-3-backend and submit the requested URLs

Deadline: April 29, 2026, 11:59pm.

And no, we are not extending it. :cy_batmanhmmm:
