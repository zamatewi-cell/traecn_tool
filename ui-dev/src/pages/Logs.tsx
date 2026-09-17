export default function Logs() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-white">系统日志</h1>
        <div className="flex items-center space-x-2">
          <select className="px-3 py-2 bg-slate-700 border border-slate-600 text-slate-300 text-sm rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">
            <option>所有级别</option>
            <option>INFO</option>
            <option>WARN</option>
            <option>ERROR</option>
          </select>
          <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 text-sm font-medium rounded-lg transition-colors">
            导出日志
          </button>
        </div>
      </div>

      <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
        <p className="text-slate-400">日志页面 - 开发中...</p>
      </div>
    </div>
  );
}
