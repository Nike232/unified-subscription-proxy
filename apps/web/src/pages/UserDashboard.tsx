export default function UserDashboard() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-slate-800">Usage Overview</h2>
        <button className="px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium hover:bg-blue-600 transition-colors shadow-sm">
          Generate API Key
        </button>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">Total Requests</p>
          <div className="text-2xl font-bold text-slate-700">2,345</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">Tokens Used</p>
          <div className="text-2xl font-bold text-slate-700">452k</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">Balance</p>
          <div className="text-2xl font-bold text-emerald-500">$45.00</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">Active Keys</p>
          <div className="text-2xl font-bold text-slate-700">2</div>
        </div>
      </div>
      
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl shadow-sm border border-slate-100 p-6 min-h-[300px] flex items-center justify-center">
          <p className="text-slate-400">Token Usage Trend (Chart Placeholder)</p>
        </div>
        <div className="bg-white rounded-xl shadow-sm border border-slate-100 p-6 min-h-[300px] flex items-center justify-center">
          <p className="text-slate-400">Model Distribution (Chart Placeholder)</p>
        </div>
      </div>
    </div>
  );
}
