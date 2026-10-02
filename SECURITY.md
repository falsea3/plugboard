# Security

Plugboard holds database passwords and talks to production servers, so security
reports are welcome and handled first.

Please report a vulnerability privately through GitHub's **Report a
vulnerability** button on the repository's Security tab, not in a public issue.
Include what you found, how to reproduce it, and the Plugboard version
(`plugboard --version` or Settings ▸ About).

What counts, for example:

- a secret (database password, SSH password or key passphrase) written to disk
  outside the system keychain, or sent anywhere but its own server;
- a statement that gets past a read-only connection;
- SQL injection through table names, filters or grid edits;
- an SSH host key that is accepted when it should be refused.
