# SSL Certificates

This directory should contain your SSL certificates for HTTPS configuration.

## Required Files

For production deployment, place the following files in this directory:

- `cert.pem` - Your SSL certificate
- `key.pem` - Your private key
- `chain.pem` - Certificate chain (if applicable)

## Development

For development, you can generate self-signed certificates:

```bash
# Generate self-signed certificate (development only)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

## Production

For production, obtain certificates from a trusted Certificate Authority like:
- Let's Encrypt (free)
- Cloudflare
- Your hosting provider

## Security Notes

- Never commit actual certificate files to version control
- Ensure proper file permissions (600 for private keys)
- Regularly renew certificates before expiration