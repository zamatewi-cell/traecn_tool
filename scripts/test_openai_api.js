#!/usr/bin/env node

/**
 * OpenAI API 兼容性测试脚本
 * 
 * 用法:
 *   node test_openai_api.js
 * 
 * 前提条件:
 *   1. 启动 trae-proxy: go run cmd/trae-proxy/main.go
 *   2. 安装依赖：npm install openai
 */

const OpenAI = require('openai');

// 配置
const BASE_URL = 'http://localhost:8080/v1';
const API_KEY = 'sk-test-key'; // 任意非空字符串

// 创建客户端
const client = new OpenAI({
  baseURL: BASE_URL,
  apiKey: API_KEY
});

// 颜色输出
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m'
};

function log(color, message) {
  console.log(`${color}${message}${colors.reset}`);
}

// 测试 1: 获取模型列表
async function testModels() {
  log(colors.cyan, '\n? 测试 1: 获取模型列表');
  log(colors.blue, 'GET /v1/models');
  
  try {
    const models = await client.models.list();
    log(colors.green, '? 成功获取模型列表:');
    models.data.forEach(model => {
      console.log(`  - ${model.id} (${model.owned_by})`);
    });
    return true;
  } catch (error) {
    log(colors.red, `? 失败: ${error.message}`);
    return false;
  }
}

// 测试 2: 非流式聊天
async function testChatNonStream() {
  log(colors.cyan, '\n? 测试 2: 非流式聊天');
  log(colors.blue, 'POST /v1/chat/completions (stream=false)');
  
  try {
    const response = await client.chat.completions.create({
      model: 'deepseek-v3.1-terminus',
      messages: [
        { role: 'system', content: 'You are a helpful assistant.' },
        { role: 'user', content: 'Hello, world!' }
      ],
      temperature: 0.7,
      max_tokens: 100
    });
    
    log(colors.green, '? 成功收到响应:');
    console.log(`  Model: ${response.model}`);
    console.log(`  Usage: ${response.usage.total_tokens} tokens`);
    console.log(`  Content: ${response.choices[0].message.content.substring(0, 100)}...`);
    return true;
  } catch (error) {
    log(colors.red, `? 失败: ${error.message}`);
    return false;
  }
}

// 测试 3: 流式聊天
async function testChatStream() {
  log(colors.cyan, '\n? 测试 3: 流式聊天');
  log(colors.blue, 'POST /v1/chat/completions (stream=true)');
  
  try {
    const stream = await client.chat.completions.create({
      model: 'deepseek-v3.1-terminus',
      messages: [
        { role: 'user', content: 'Count from 1 to 5' }
      ],
      stream: true
    });
    
    log(colors.green, '? 开始接收流式响应:');
    process.stdout.write('  ');
    
    for await (const chunk of stream) {
      const content = chunk.choices[0]?.delta?.content || '';
      if (content) {
        process.stdout.write(content);
      }
    }
    
    console.log('\n  ? 流式传输完成');
    return true;
  } catch (error) {
    log(colors.red, `? 失败: ${error.message}`);
    return false;
  }
}

// 测试 4: 健康检查
async function testHealth() {
  log(colors.cyan, '\n?? 测试 4: 健康检查');
  log(colors.blue, 'GET /health');
  
  try {
    const response = await fetch('http://localhost:8080/health');
    const data = await response.json();
    
    if (data.status === 'ok') {
      log(colors.green, `? 服务健康: ${data.status} (v${data.version})`);
      return true;
    } else {
      log(colors.red, `? 服务状态异常: ${data.status}`);
      return false;
    }
  } catch (error) {
    log(colors.red, `? 失败: ${error.message}`);
    return false;
  }
}

// 测试 5: 队列状态
async function testQueueStatus() {
  log(colors.cyan, '\n? 测试 5: 队列状态');
  log(colors.blue, 'GET /v1/queue/status');
  
  try {
    const response = await fetch('http://localhost:8080/v1/queue/status');
    const data = await response.json();
    
    log(colors.green, '? 队列状态:');
    console.log(`  Queue Length: ${data.queue_length || 'N/A'}`);
    console.log(`  Active Requests: ${data.active_requests || 'N/A'}`);
    console.log(`  Estimated Wait: ${data.estimated_wait_seconds || 0}s`);
    return true;
  } catch (error) {
    log(colors.red, `? 失败: ${error.message}`);
    return false;
  }
}

// 主函数
async function main() {
  log(colors.yellow, '\n========================================');
  log(colors.yellow, '   Trae CN Proxy - OpenAI API 测试');
  log(colors.yellow, '========================================\n');
  
  const results = {
    models: await testModels(),
    health: await testHealth(),
    queue: await testQueueStatus(),
    chatNonStream: await testChatNonStream(),
    chatStream: await testChatStream()
  };
  
  // 统计结果
  log(colors.yellow, '\n========================================');
  log(colors.yellow, '   测试结果汇总');
  log(colors.yellow, '========================================\n');
  
  const passed = Object.values(results).filter(r => r).length;
  const total = Object.values(results).length;
  
  log(colors.blue, `通过：${passed}/${total}`);
  
  if (passed === total) {
    log(colors.green, '\n? 所有测试通过！');
  } else {
    log(colors.red, `\n??  ${total - passed} 个测试失败`);
  }
  
  console.log('');
}

// 运行测试
main().catch(error => {
  log(colors.red, `\n? 测试执行出错: ${error.message}`);
  process.exit(1);
});
