// Trae CN 认证流程实现
// 用于演示完整的认证和 Token 刷新机制

const fs = require('fs');
const https = require('https');
const { parseJwt } = require('./parse_token');

class TraeAuth {
  constructor() {
    this.storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
    this.token = null;
    this.refreshToken = null;
    this.expiredAt = null;
    this.refreshExpiredAt = null;
    this.userId = null;
    this.host = 'https://api.trae.com.cn';
  }

  // 从存储加载认证信息
  loadFromStorage() {
    try {
      const storage = JSON.parse(fs.readFileSync(this.storagePath, 'utf8'));
      const authInfo = JSON.parse(storage['iCubeAuthInfo://icube.cloudide']);
      
      this.token = authInfo.token;
      this.refreshToken = authInfo.refreshToken;
      this.expiredAt = new Date(authInfo.expiredAt);
      this.refreshExpiredAt = new Date(authInfo.refreshExpiredAt);
      this.userId = authInfo.userId;
      this.host = authInfo.host;
      
      console.log('? 从存储加载认证信息成功');
      console.log(`   用户 ID: ${this.userId}`);
      console.log(`   Token 过期时间：${this.expiredAt.toISOString()}`);
      console.log(`   Refresh Token 过期时间：${this.refreshExpiredAt.toISOString()}`);
      
      return true;
    } catch (error) {
      console.error('? 加载认证信息失败:', error.message);
      return false;
    }
  }

  // 检查 Token 是否过期
  isTokenExpired() {
    if (!this.expiredAt) {
      return true;
    }
    const now = new Date();
    // 提前 5 分钟判定为过期
    return now >= new Date(this.expiredAt.getTime() - 5 * 60 * 1000);
  }

  // 检查 Refresh Token 是否过期
  isRefreshTokenExpired() {
    if (!this.refreshExpiredAt) {
      return true;
    }
    const now = new Date();
    return now >= this.refreshExpiredAt;
  }

  // 获取有效的 Token（自动刷新）
  async getValidToken() {
    if (!this.loadFromStorage()) {
      throw new Error('无法加载认证信息，请先登录');
    }

    if (this.isRefreshTokenExpired()) {
      throw new Error('Refresh Token 已过期，请重新登录');
    }

    if (this.isTokenExpired()) {
      console.log('??  Token 已过期，正在刷新...');
      await this.refreshToken();
    }

    return this.token;
  }

  // 刷新 Token
  async refreshToken() {
    console.log('? 开始刷新 Token...');

    const url = `${this.host}/api/auth/refresh_token`;
    
    const headers = {
      'Content-Type': 'application/json',
      'User-Agent': 'TraeClient/TTNet',
      'X-Device-Id': this.getDeviceId(),
      'X-Machine-Id': this.getMachineId(),
      'X-Request-ID': this.generateRequestId(),
    };

    const body = {
      refresh_token: this.refreshToken,
      user_id: this.userId
    };

    try {
      const response = await this.makeRequest('POST', url, headers, body);
      
      if (response.token) {
        this.token = response.token;
        this.expiredAt = new Date(response.expiredAt);
        console.log('? Token 刷新成功');
        console.log(`   新 Token 过期时间：${this.expiredAt.toISOString()}`);
        
        // 保存到存储
        this.saveToStorage();
        
        return this.token;
      } else {
        throw new Error('刷新响应中缺少 Token');
      }
    } catch (error) {
      console.error('? Token 刷新失败:', error.message);
      throw error;
    }
  }

  // 发送认证请求
  async makeAuthenticatedRequest(method, endpoint, body = null) {
    const token = await this.getValidToken();
    
    const url = `${this.host}${endpoint}`;
    const headers = {
      'Content-Type': 'application/json',
      'User-Agent': 'TraeClient/TTNet',
      'X-Ide-Token': token,
      'X-Device-Id': this.getDeviceId(),
      'X-Machine-Id': this.getMachineId(),
      'X-Request-ID': this.generateRequestId(),
      'X-Custom-Trace-Id': this.generateTraceId(),
    };

    return await this.makeRequest(method, url, headers, body);
  }

  // 发送 HTTP 请求
  makeRequest(method, url, headers, body = null) {
    return new Promise((resolve, reject) => {
      const urlObj = new URL(url);
      const options = {
        hostname: urlObj.hostname,
        port: 443,
        path: urlObj.pathname + urlObj.search,
        method: method,
        headers: headers
      };

      const req = https.request(options, (res) => {
        let data = '';
        
        res.on('data', (chunk) => {
          data += chunk;
        });
        
        res.on('end', () => {
          if (res.statusCode >= 200 && res.statusCode < 300) {
            try {
              resolve(JSON.parse(data));
            } catch (e) {
              resolve(data);
            }
          } else {
            reject(new Error(`HTTP ${res.statusCode}: ${data}`));
          }
        });
      });

      req.on('error', (error) => {
        reject(error);
      });

      if (body) {
        req.write(JSON.stringify(body));
      }
      
      req.end();
    });
  }

  // 获取设备 ID
  getDeviceId() {
    // 从存储或生成设备 ID
    return '2262131830954826';
  }

  // 获取机器 ID
  getMachineId() {
    // 从存储或生成机器 ID
    return '323072c1635648e2034a597de45cecfb28ee6fd5d74339197d47c77336d36ca3';
  }

  // 生成请求 ID
  generateRequestId() {
    return `req_${this.generateUUID()}`;
  }

  // 生成追踪 ID
  generateTraceId() {
    return this.generateUUID();
  }

  // 生成 UUID
  generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
      const r = Math.random() * 16 | 0;
      const v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }

  // 保存认证信息到存储
  saveToStorage() {
    try {
      const storage = JSON.parse(fs.readFileSync(this.storagePath, 'utf8'));
      const authInfo = JSON.parse(storage['iCubeAuthInfo://icube.cloudide']);
      
      authInfo.token = this.token;
      authInfo.expiredAt = this.expiredAt.toISOString();
      
      storage['iCubeAuthInfo://icube.cloudide'] = JSON.stringify(authInfo);
      fs.writeFileSync(this.storagePath, JSON.stringify(storage, null, 2));
      
      console.log('? 认证信息已保存到存储');
    } catch (error) {
      console.error('? 保存认证信息失败:', error.message);
    }
  }

  // 测试认证流程
  async testAuth() {
    console.log('=== 认证流程测试 ===\n');
    
    try {
      // 1. 加载认证信息
      if (!this.loadFromStorage()) {
        console.log('? 测试失败：无法加载认证信息');
        return;
      }

      // 2. 检查 Token 状态
      console.log('\n? Token 状态检查:');
      console.log(`   Token 是否过期：${this.isTokenExpired() ? '是' : '否'}`);
      console.log(`   Refresh Token 是否过期：${this.isRefreshTokenExpired() ? '是' : '否'}`);

      // 3. 解析 Token
      console.log('\n? Token 解析:');
      const parsed = parseJwt(this.token);
      if (parsed) {
        console.log(`   用户 ID: ${parsed.payload.data.id}`);
        console.log(`   租户 ID: ${parsed.payload.data.tenant_id}`);
        console.log(`   签发时间：${new Date(parsed.payload.iat * 1000).toISOString()}`);
        console.log(`   过期时间：${new Date(parsed.payload.exp * 1000).toISOString()}`);
      }

      // 4. 测试获取有效 Token
      console.log('\n? 获取有效 Token:');
      const validToken = await this.getValidToken();
      console.log(`   ? 获取成功，Token 长度：${validToken.length}`);

      // 5. 测试认证请求
      console.log('\n? 测试认证请求:');
      const result = await this.makeAuthenticatedRequest('GET', '/api/user/profile');
      console.log('   ? 请求成功');
      console.log('   响应:', JSON.stringify(result, null, 2).substring(0, 200) + '...');

      console.log('\n? 认证流程测试完成\n');
    } catch (error) {
      console.error('\n? 测试失败:', error.message);
      console.error(error.stack);
    }
  }
}

// 命令行执行
if (require.main === module) {
  const auth = new TraeAuth();
  
  const args = process.argv.slice(2);
  const command = args[0] || 'test';

  switch (command) {
    case 'test':
      auth.testAuth();
      break;
    case 'token':
      auth.loadFromStorage();
      console.log('\n当前 Token:');
      console.log(auth.token);
      break;
    case 'parse':
      auth.loadFromStorage();
      const parsed = parseJwt(auth.token);
      console.log('\n解析结果:');
      console.log(JSON.stringify(parsed, null, 2));
      break;
    case 'refresh':
      auth.loadFromStorage();
      auth.refreshToken().catch(console.error);
      break;
    default:
      console.log('用法：node auth_flow.js [test|token|parse|refresh]');
      console.log('  test   - 测试完整认证流程');
      console.log('  token  - 显示当前 Token');
      console.log('  parse  - 解析 Token');
      console.log('  refresh - 刷新 Token');
  }
}

// 导出模块
module.exports = TraeAuth;
