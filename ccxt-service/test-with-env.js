// Test with proper environment variables set
process.env.ADMIN_API_KEY = 'test-admin-key-that-is-at-least-32-characters-long-for-security';
process.env.PORT = '3002';

// Now import the service
const mod = await import('./index.ts');
const service = mod.default;

console.log('Service loaded successfully');
console.log('Service port:', service.port);

// Test the exchanges endpoint
const res = await service.fetch(new Request('http://localhost/api/exchanges'));
const body = await res.json();
console.log('Exchanges response status:', res.status);
console.log('Exchanges count:', body.exchanges?.length || 0);
console.log('First few exchanges:', body.exchanges?.slice(0, 5) || []);
console.log('Contains binance:', body.exchanges?.includes('binance') || false);