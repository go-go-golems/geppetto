// Package qwen3 implements Qwen3-Reranker scoring through llama.cpp's
// /completion endpoint.
//
// Qwen3-Reranker is a causal language model rather than a BERT-style
// cross-encoder. Each query/document pair is rendered with Qwen's official
// judgment template, one token is generated, and relevance is the normalized
// probability of "yes" versus "no". The provider bounds pair concurrency and
// implements the transport-neutral rerank.Provider contract.
package qwen3
