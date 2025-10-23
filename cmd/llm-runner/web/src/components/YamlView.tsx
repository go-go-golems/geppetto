import { useState } from 'react';
import yaml from 'js-yaml';
import { Copy, Check } from 'lucide-react';

interface Props {
  data: any;
  title?: string;
  maxHeight?: string;
}

export default function YamlView({ data, title, maxHeight = '400px' }: Props) {
  const [copied, setCopied] = useState(false);

  const yamlString = yaml.dump(data, {
    indent: 2,
    lineWidth: 120,
    noRefs: true,
  });

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(yamlString);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <div className="relative">
      {title && (
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium text-gray-300">{title}</span>
        </div>
      )}
      <div className="relative group">
        <button
          onClick={handleCopy}
          className="absolute top-2 right-2 p-2 bg-gray-700 hover:bg-gray-600 rounded text-gray-300 hover:text-white transition-colors opacity-0 group-hover:opacity-100 z-10"
          title="Copy to clipboard"
        >
          {copied ? <Check size={16} /> : <Copy size={16} />}
        </button>
        <pre
          className="p-3 bg-gray-900 rounded overflow-x-auto text-xs text-gray-200"
          style={{ maxHeight }}
        >
          {yamlString}
        </pre>
      </div>
    </div>
  );
}

