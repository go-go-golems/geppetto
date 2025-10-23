export interface ArtifactRun {
  id: string;
  path: string;
  timestamp: number;
  hasTurns: boolean;
  hasEvents: boolean;
  hasLogs: boolean;
  hasRaw: boolean;
  turnCount: number;
}

export interface Turn {
  id: string;
  blocks: Block[];
  executionIndex: number;
  label: string;
  rawYaml?: string;
  rawRequestIndex?: number;
  metadata?: Record<string, any>;
}

export interface Block {
  kind: string;
  role: string;
  payload: {
    text?: string;
    [key: string]: any;
  };
}

export interface Event {
  type: string;
  timestamp: number;
  data: Record<string, any>;
  meta?: {
    model?: string;
    message_id?: string;
    turn_id?: string;
    [key: string]: any;
  };
}

export interface LogEntry {
  level: string;
  time: string;
  message: string;
  error?: string;
  [key: string]: any;
}

export interface RawArtifact {
  turnIndex: number;
  inputTurnIndex: number;
  inputTurnYaml?: string;
  httpRequest?: HttpRequest;
  httpResponse?: HttpResponse;
  sseLog?: string;
  providerObjects: ProviderObject[];
}

export interface HttpRequest {
  turn_index: number;
  turn_id: string;
  method: string;
  url: string;
  headers: Record<string, string[]>;
  body: string;
}

export interface HttpResponse {
  turn_index: number;
  turn_id: string;
  status: number;
  headers: Record<string, string[]>;
  body: any;
}

export interface ProviderObject {
  sequence: number;
  type: string;
  data: any;
}

export interface ParsedRun {
  id: string;
  path: string;
  inputTurn?: Turn;
  turns: Turn[];
  events: Event[][];
  logs: LogEntry[];
  raw: RawArtifact[];
  errors: ErrorContext[];
}

export interface ErrorContext {
  turnIndex: number;
  error: string;
  relatedLogs: LogEntry[];
  relatedEvents: Event[];
  httpRequest?: HttpRequest;
  httpResponse?: HttpResponse;
}

