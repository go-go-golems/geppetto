import { useState } from 'react';
import type { ParsedRun, Turn } from '../types';
import { ChevronDown, ChevronRight, User, Bot, Settings, ExternalLink } from 'lucide-react';
import YamlView from './YamlView';
import CopyButton from './CopyButton';
import type { TabType } from './RunView';

interface Props {
  run: ParsedRun;
  onNavigate?: (tab: TabType, turnIndex?: number) => void;
}

export default function TurnTimeline({ run, onNavigate }: Props) {
  const [expandedTurns, setExpandedTurns] = useState<Set<number>>(new Set([0]));
  const [expandedBlocks, setExpandedBlocks] = useState<Set<string>>(new Set());

  const toggleTurn = (index: number) => {
    const newSet = new Set(expandedTurns);
    if (newSet.has(index)) {
      newSet.delete(index);
    } else {
      newSet.add(index);
    }
    setExpandedTurns(newSet);
  };

  const toggleBlock = (key: string) => {
    const newSet = new Set(expandedBlocks);
    if (newSet.has(key)) {
      newSet.delete(key);
    } else {
      newSet.add(key);
    }
    setExpandedBlocks(newSet);
  };

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'user':
        return <User size={16} className="text-blue-400" />;
      case 'assistant':
        return <Bot size={16} className="text-green-400" />;
      case 'system':
        return <Settings size={16} className="text-purple-400" />;
      default:
        return null;
    }
  };

  const getRoleColor = (role: string) => {
    switch (role) {
      case 'user':
        return 'border-blue-500 bg-blue-900/20';
      case 'assistant':
        return 'border-green-500 bg-green-900/20';
      case 'system':
        return 'border-purple-500 bg-purple-900/20';
      default:
        return 'border-gray-500 bg-gray-900/20';
    }
  };

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="max-w-4xl mx-auto space-y-4">
        {/* Input Turn */}
        {run.inputTurn && (
          <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-3">
              <span className="px-2 py-1 text-xs bg-gray-700 text-gray-300 rounded">
                {run.inputTurn.label || 'Input Turn'}
              </span>
              <span className="text-sm text-gray-400">{run.inputTurn.id}</span>
            </div>
            <TurnBlocks
              turn={run.inputTurn}
              expandedBlocks={expandedBlocks}
              toggleBlock={toggleBlock}
              getRoleIcon={getRoleIcon}
              getRoleColor={getRoleColor}
              keyPrefix="input"
            />
            
            {run.inputTurn.rawYaml && (
              <details className="mt-4">
                <summary className="cursor-pointer text-sm font-medium text-gray-400 hover:text-gray-300">
                  Show Raw Turn YAML
                </summary>
                <div className="mt-2 relative group">
                  <CopyButton 
                    text={run.inputTurn.rawYaml} 
                    className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 z-10"
                  />
                  <pre className="p-3 bg-gray-900 rounded overflow-x-auto text-xs text-gray-200 max-h-96 overflow-y-auto">
                    {run.inputTurn.rawYaml}
                  </pre>
                </div>
              </details>
            )}
          </div>
        )}

        {/* Turns */}
        {run.turns.map((turn, index) => (
          <div key={index} className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden">
            <div
              className="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-750"
              onClick={() => toggleTurn(index)}
            >
              <div className="flex items-center gap-3 flex-wrap flex-1">
                {expandedTurns.has(index) ? (
                  <ChevronDown size={20} className="text-gray-400" />
                ) : (
                  <ChevronRight size={20} className="text-gray-400" />
                )}
                <span className="font-semibold text-white">
                  {turn.label || `Turn ${index + 1}`}
                </span>
                <span className="text-sm text-gray-400">{turn.id}</span>
                <span className="px-2 py-1 text-xs bg-gray-700 text-gray-300 rounded">
                  {turn.blocks.length} block{turn.blocks.length !== 1 ? 's' : ''}
                </span>
                {turn.executionIndex >= 0 && (
                  <span className="px-2 py-1 text-xs bg-blue-900 text-blue-200 rounded">
                    Step {turn.executionIndex}
                  </span>
                )}
              </div>
              {onNavigate && turn.rawRequestIndex !== undefined && (
                <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                  <button
                    onClick={() => onNavigate('events', index)}
                    className="px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-600 rounded flex items-center gap-1"
                    title="View events for this turn"
                  >
                    <ExternalLink size={12} />
                    Events
                  </button>
                  <button
                    onClick={() => onNavigate('raw', turn.rawRequestIndex)}
                    className="px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-600 rounded flex items-center gap-1"
                    title="View raw HTTP request/response"
                  >
                    <ExternalLink size={12} />
                    Raw
                  </button>
                </div>
              )}
            </div>

            {expandedTurns.has(index) && (
              <div className="p-4 pt-0 border-t border-gray-700">
                <TurnBlocks
                  turn={turn}
                  expandedBlocks={expandedBlocks}
                  toggleBlock={toggleBlock}
                  getRoleIcon={getRoleIcon}
                  getRoleColor={getRoleColor}
                  keyPrefix={`turn-${index}`}
                />
                
                {turn.rawYaml && (
                  <details className="mt-4" open={turn.rawRequestIndex !== undefined}>
                    <summary className="cursor-pointer text-sm font-medium text-gray-400 hover:text-gray-300">
                      Show Raw Turn YAML {turn.rawRequestIndex !== undefined && '→ API Request Transformation'}
                    </summary>
                    <div className="mt-2 space-y-4">
                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-xs font-semibold text-gray-400">Turn YAML (Serialized)</span>
                        </div>
                        <div className="relative group">
                          <CopyButton 
                            text={turn.rawYaml} 
                            className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 z-10"
                          />
                          <pre className="p-3 bg-gray-900 rounded overflow-x-auto text-xs text-gray-200 max-h-80 overflow-y-auto border-l-4 border-blue-500">
                            {turn.rawYaml}
                          </pre>
                        </div>
                      </div>
                      
                      {turn.rawRequestIndex !== undefined && run.raw[turn.rawRequestIndex] && (
                        <div>
                          <div className="flex items-center gap-2 mb-2">
                            <span className="text-xs font-semibold text-gray-400">↓ Transformed to API Request Body</span>
                            <span className="px-2 py-0.5 text-xs bg-orange-900 text-orange-200 rounded">
                              Turn {run.raw[turn.rawRequestIndex].turnIndex}
                            </span>
                          </div>
                          {run.raw[turn.rawRequestIndex].httpRequest && (
                            <YamlView 
                              data={typeof run.raw[turn.rawRequestIndex].httpRequest!.body === 'string'
                                ? JSON.parse(run.raw[turn.rawRequestIndex].httpRequest!.body)
                                : run.raw[turn.rawRequestIndex].httpRequest!.body}
                              maxHeight="400px"
                            />
                          )}
                        </div>
                      )}
                    </div>
                  </details>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

interface TurnBlocksProps {
  turn: Turn;
  expandedBlocks: Set<string>;
  toggleBlock: (key: string) => void;
  getRoleIcon: (role: string) => React.ReactNode;
  getRoleColor: (role: string) => string;
  keyPrefix: string;
}

function TurnBlocks({
  turn,
  expandedBlocks,
  toggleBlock,
  getRoleIcon,
  getRoleColor,
  keyPrefix,
}: TurnBlocksProps) {
  return (
    <div className="space-y-2">
      {turn.blocks.map((block, blockIndex) => {
        const blockKey = `${keyPrefix}-block-${blockIndex}`;
        const isExpanded = expandedBlocks.has(blockKey);
        const text = block.payload?.text || '';
        const truncated = text.length > 200 ? text.slice(0, 200) + '...' : text;

        return (
          <div
            key={blockIndex}
            className={`border-l-4 rounded-r-lg p-3 ${getRoleColor(block.role)}`}
          >
            <div className="flex items-start justify-between gap-2 mb-2">
              <div className="flex items-center gap-2">
                {getRoleIcon(block.role)}
                <span className="text-sm font-medium text-gray-300 capitalize">
                  {block.role}
                </span>
                {block.kind !== block.role && (
                  <span className="text-xs text-gray-500">({block.kind})</span>
                )}
              </div>
              {text.length > 200 && (
                <button
                  onClick={() => toggleBlock(blockKey)}
                  className="text-xs text-blue-400 hover:text-blue-300"
                >
                  {isExpanded ? 'Show less' : 'Show more'}
                </button>
              )}
            </div>
            <div className="text-sm text-gray-200 whitespace-pre-wrap font-mono relative group">
              {isExpanded ? text : truncated}
              {text && (
                <CopyButton 
                  text={text} 
                  className="absolute top-0 right-0 opacity-0 group-hover:opacity-100"
                />
              )}
            </div>
            {Object.keys(block.payload).length > 1 && (
              <details className="mt-2 text-xs">
                <summary className="cursor-pointer text-gray-500 hover:text-gray-400">
                  Additional payload data
                </summary>
                <div className="mt-2">
                  <YamlView data={block.payload} maxHeight="300px" />
                </div>
              </details>
            )}
          </div>
        );
      })}
    </div>
  );
}

