import type { ParsedRun } from '../types';
import { AlertCircle, AlertTriangle, FileText, Activity, Network } from 'lucide-react';
import YamlView from './YamlView';
import CopyButton from './CopyButton';

interface Props {
  run: ParsedRun;
  onNavigate?: (tab: any, turnIndex?: number) => void;
}

export default function ErrorContext({ run }: Props) {
  if (run.errors.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center text-gray-400">
          <AlertCircle size={48} className="mx-auto mb-4 opacity-50 text-green-500" />
          <p className="text-green-400">No errors found!</p>
          <p className="text-sm mt-2">All runs completed successfully</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="max-w-5xl mx-auto space-y-6">
        {run.errors.map((error, index) => (
          <div key={index} className="bg-gray-800 border-2 border-red-500 rounded-lg overflow-hidden">
            {/* Error Header */}
            <div className="bg-red-900/30 p-4 border-b border-red-500">
              <div className="flex items-start gap-3">
                <AlertCircle size={24} className="text-red-400 flex-shrink-0 mt-1" />
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <h3 className="text-lg font-semibold text-white">Error in Turn {error.turnIndex + 1}</h3>
                    <span className="px-2 py-1 text-xs bg-red-600 text-white rounded">
                      ERROR
                    </span>
                  </div>
                  <div className="text-sm text-red-200 font-mono bg-red-950/50 p-3 rounded">
                    {error.error}
                  </div>
                </div>
              </div>
            </div>

            {/* Error Details */}
            <div className="p-4 space-y-4">
              {/* Related Logs */}
              {error.relatedLogs.length > 0 && (
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <FileText size={16} className="text-purple-400" />
                    <h4 className="font-semibold text-white">
                      Related Logs ({error.relatedLogs.length})
                    </h4>
                  </div>
                  <div className="space-y-2">
                    {error.relatedLogs.map((log, logIndex) => (
                      <div
                        key={logIndex}
                        className="border-l-4 border-red-500 bg-red-900/10 rounded-r-lg p-3"
                      >
                        <div className="flex items-center justify-between mb-2">
                          <span className="px-2 py-1 text-xs bg-red-600 text-white rounded uppercase font-semibold">
                            {log.level}
                          </span>
                          <span className="text-xs text-gray-400 font-mono">{log.time}</span>
                        </div>
                  <div className="text-sm text-white mb-2 flex items-start justify-between">
                    <span>{log.message}</span>
                    <CopyButton text={log.message} />
                  </div>
                  {log.error && (
                    <div className="relative group">
                      <CopyButton 
                        text={log.error} 
                        className="absolute top-1 right-1 opacity-0 group-hover:opacity-100"
                      />
                      <div className="text-xs text-red-300 font-mono bg-red-950/50 p-2 rounded">
                        {log.error}
                      </div>
                    </div>
                  )}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Related Events */}
              {error.relatedEvents.length > 0 && (
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Activity size={16} className="text-blue-400" />
                    <h4 className="font-semibold text-white">
                      Related Events ({error.relatedEvents.length})
                    </h4>
                  </div>
                  <div className="space-y-1 max-h-48 overflow-y-auto">
                    {error.relatedEvents.map((event, eventIndex) => (
                      <div
                        key={eventIndex}
                        className="flex items-center justify-between text-sm p-2 bg-gray-900/50 rounded"
                      >
                        <div className="flex items-center gap-2">
                          <span className="px-2 py-0.5 text-xs bg-blue-600 text-white rounded font-mono">
                            {event.type}
                          </span>
                          {event.meta?.message_id && (
                            <span className="text-xs text-gray-400">
                              {event.meta.message_id.slice(0, 8)}
                            </span>
                          )}
                        </div>
                        <span className="text-xs text-gray-500">
                          {new Date(event.timestamp).toLocaleTimeString()}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* HTTP Request */}
              {error.httpRequest && (
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Network size={16} className="text-orange-400" />
                    <h4 className="font-semibold text-white">HTTP Request</h4>
                  </div>
                  <div className="bg-gray-900/50 rounded-lg p-3 space-y-2">
                    <div className="flex items-center gap-2">
                      <span className="px-2 py-1 text-xs bg-blue-600 text-white rounded font-semibold">
                        {error.httpRequest.method}
                      </span>
                      <span className="text-sm text-gray-300 font-mono">
                        {error.httpRequest.url}
                      </span>
                    </div>
                    <details>
                      <summary className="cursor-pointer text-sm text-gray-400 hover:text-gray-300">
                        Request Body
                      </summary>
                      <div className="mt-2">
                        <YamlView 
                          data={typeof error.httpRequest.body === 'string'
                            ? JSON.parse(error.httpRequest.body)
                            : error.httpRequest.body}
                          maxHeight="300px"
                        />
                      </div>
                    </details>
                  </div>
                </div>
              )}

              {/* HTTP Response */}
              {error.httpResponse && (
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Network size={16} className="text-orange-400" />
                    <h4 className="font-semibold text-white">HTTP Response</h4>
                  </div>
                  <div className="bg-gray-900/50 rounded-lg p-3 space-y-2">
                    <div className="flex items-center gap-2">
                      <span
                        className={`px-2 py-1 text-xs rounded font-semibold ${
                          error.httpResponse.status >= 200 && error.httpResponse.status < 300
                            ? 'bg-green-600 text-white'
                            : 'bg-red-600 text-white'
                        }`}
                      >
                        {error.httpResponse.status}
                      </span>
                    </div>
                    <details open>
                      <summary className="cursor-pointer text-sm text-gray-400 hover:text-gray-300">
                        Response Body
                      </summary>
                      <div className="mt-2">
                        <YamlView data={error.httpResponse.body} maxHeight="300px" />
                      </div>
                    </details>
                  </div>
                </div>
              )}

              {/* Troubleshooting Hints */}
              <div className="bg-yellow-900/20 border border-yellow-600 rounded-lg p-3">
                <div className="flex items-start gap-2">
                  <AlertTriangle size={16} className="text-yellow-400 flex-shrink-0 mt-0.5" />
                  <div>
                    <div className="text-sm font-semibold text-yellow-200 mb-1">
                      Troubleshooting Tips
                    </div>
                    <ul className="text-xs text-yellow-100 space-y-1">
                      <li>• Check the HTTP request/response for malformed data</li>
                      <li>• Review related events for sequence issues</li>
                      <li>• Verify the model configuration and parameters</li>
                      <li>• Check if rate limits or quotas were exceeded</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

