const d = require('./detail_param_chat_v3.json');
const eps = d.metadata.encrypted_prompt_set;

// Get the smallest prompt (single_chat master_agent)
const p = eps['6'];
const buf = Buffer.from(p.prompt_template, 'base64');
console.log('Decoded bytes:', buf.length);
console.log('First 32 hex:', buf.subarray(0, 32).toString('hex'));

// Try XOR known-plaintext
const guesses = [
  'You are', '# System', 'You are a ', '<|system|>',
  'Act as', '## Instructions', '# Role',
  '{"system', '{"role":', 'system_prompt',
];

for (const guess of guesses) {
  const key = Buffer.alloc(guess.length);
  for (let i = 0; i < guess.length; i++) {
    key[i] = buf[i] ^ guess.charCodeAt(i);
  }
  const hex = key.toString('hex');
  const ascii = [...key].map(b => (b >= 0x20 && b < 0x7f) ? String.fromCharCode(b) : '.').join('');
  console.log(`"${guess}" => key: ${hex} (${ascii})`);
}

// Also check if it could be AES - look for repeating patterns
// Check if multiple prompts share the same first bytes (same IV/nonce)
console.log('\n=== First 16 bytes of each prompt ===');
for (let i = 0; i < 14; i++) {
  const pb = Buffer.from(eps[String(i)].prompt_template, 'base64');
  console.log(`[${i}] ${pb.subarray(0, 16).toString('hex')} key=${eps[String(i)].prompt_key.split('.').pop()}`);
}

// Check if there's a common XOR key by XORing two prompts  
// If both are XOR'd with same key, XOR of two encryptions = XOR of two plaintexts
const p0 = Buffer.from(eps['6'].prompt_template, 'base64');
const p1 = Buffer.from(eps['3'].prompt_template, 'base64');
const xored = Buffer.alloc(Math.min(64, p0.length, p1.length));
for (let i = 0; i < xored.length; i++) {
  xored[i] = p0[i] ^ p1[i];
}
console.log('\nXOR of prompt[6] ^ prompt[3] (first 64 bytes):');
console.log('Hex:', xored.toString('hex'));
// If it's XOR with same key, this should look like XOR of two text docs
// Check if it has many printable chars
const printable = [...xored].filter(b => b >= 0x20 && b < 0x7f).length;
console.log(`Printable chars: ${printable}/${xored.length} (${(printable/xored.length*100).toFixed(0)}%)`);
