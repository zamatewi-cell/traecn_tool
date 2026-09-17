// Token 解析工具 - 用于分析 Trae CN 的 JWT Token 结构
const fs = require('fs');

function base64UrlDecode(str) {
  // Base64URL 转 Base64
  let base64 = str.replace(/-/g, '+').replace(/_/g, '/');
  // 补齐 padding
  while (base64.length % 4) {
    base64 += '=';
  }
  return Buffer.from(base64, 'base64').toString('utf8');
}

function parseJwt(token) {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) {
      throw new Error('Invalid JWT format');
    }

    const header = JSON.parse(base64UrlDecode(parts[0]));
    const payload = JSON.parse(base64UrlDecode(parts[1]));
    const signature = parts[2];

    return {
      header,
      payload,
      signature,
      raw: {
        header: parts[0],
        payload: parts[1],
        signature: parts[2]
      }
    };
  } catch (error) {
    console.error('Failed to parse JWT:', error.message);
    return null;
  }
}

function formatTimestamp(timestamp) {
  return new Date(timestamp * 1000).toISOString();
}

// 从存储中读取 Token
function loadToken() {
  const storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
  const storage = JSON.parse(fs.readFileSync(storagePath, 'utf8'));
  const authInfo = JSON.parse(storage['iCubeAuthInfo://icube.cloudide']);
  return authInfo.token;
}

// 主函数
function analyzeToken(tokenSource = 'storage') {
  console.log('=== Trae CN Token 分析工具 ===\n');

  let token;
  if (tokenSource === 'storage') {
    console.log('从存储加载 Token...');
    token = loadToken();
  } else if (tokenSource.startsWith('eyJ')) {
    token = tokenSource;
  } else {
    console.error('请提供有效的 Token 或使用 "storage" 从存储加载');
    return;
  }

  if (!token) {
    console.error('未找到 Token');
    return;
  }

  console.log(`Token 长度：${token.length} 字符\n`);

  const parsed = parseJwt(token);
  if (!parsed) {
    return;
  }

  console.log('=== Header ===');
  console.log(JSON.stringify(parsed.header, null, 2));
  console.log('\n编码后的 Header:', parsed.raw.header);

  console.log('\n=== Payload ===');
  console.log(JSON.stringify(parsed.payload, null, 2));

  if (parsed.payload.data) {
    console.log('\n=== Data 字段详情 ===');
    console.log(`用户 ID: ${parsed.payload.data.id}`);
    console.log(`来源：${parsed.payload.data.source}`);
    console.log(`来源 ID: ${parsed.payload.data.source_id}`);
    console.log(`租户 ID: ${parsed.payload.data.tenant_id}`);
    console.log(`类型：${parsed.payload.data.type}`);
  }

  if (parsed.payload.exp) {
    console.log('\n=== 时间信息 ===');
    console.log(`签发时间：${formatTimestamp(parsed.payload.iat)}`);
    console.log(`过期时间：${formatTimestamp(parsed.payload.exp)}`);
    const now = Math.floor(Date.now() / 1000);
    const remaining = parsed.payload.exp - now;
    const daysRemaining = Math.floor(remaining / 86400);
    console.log(`剩余有效期：${daysRemaining} 天 (${remaining} 秒)`);
    
    if (daysRemaining < 3) {
      console.log('??  警告：Token 即将过期，需要刷新！');
    } else {
      console.log('? Token 状态正常');
    }
  }

  console.log('\n=== Signature ===');
  console.log(`签名算法：${parsed.header.alg}`);
  console.log(`签名长度：${parsed.signature.length} 字符`);
  console.log(`签名 (前 32 字符): ${parsed.signature.substring(0, 32)}...`);

  console.log('\n=== 完整 Token ===');
  console.log(token);
  console.log('\n=================================\n');
}

// 命令行参数处理
const args = process.argv.slice(2);
const tokenSource = args[0] || 'storage';

analyzeToken(tokenSource);

// 导出函数供其他模块使用
module.exports = {
  parseJwt,
  base64UrlDecode,
  analyzeToken,
  loadToken
};
