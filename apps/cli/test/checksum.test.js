const assert = require('assert');
const crypto = require('crypto');
const fs = require('fs');
const os = require('os');
const path = require('path');
const { verifyChecksum } = require('../scripts/postinstall.js');

const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ig-checksum-'));
const archive = path.join(dir, 'issuegraph_1.0.0_linux_amd64.tar.gz');
fs.writeFileSync(archive, 'payload');
const sum = crypto.createHash('sha256').update('payload').digest('hex');
const name = path.basename(archive);

verifyChecksum(archive, `${'0'.repeat(64)}  other.tar.gz\n${sum}  ${name}\n`, name);
assert.throws(() => verifyChecksum(archive, `${'0'.repeat(64)}  ${name}\n`, name), /mismatch/);
assert.throws(() => verifyChecksum(archive, `${sum}  other.tar.gz\n`, name), /No checksum/);

fs.rmSync(dir, { recursive: true, force: true });
console.log('✓ checksum tests passed');
