package ai

import (
	"testing"
)

func TestBuildMemoryVectorRanksRelatedChineseTextHigher(t *testing.T) {
	query := BuildMemoryVector("我喜欢吃草莓蛋糕", 256)
	related := BuildMemoryVector("用户最喜欢草莓味的生日蛋糕", 256)
	unrelated := BuildMemoryVector("明天下午去图书馆看历史书", 256)

	relatedScore := cosineSimilarity(query, related)
	unrelatedScore := cosineSimilarity(query, unrelated)
	if relatedScore <= unrelatedScore {
		t.Fatalf("expected related memory score %.3f > unrelated score %.3f", relatedScore, unrelatedScore)
	}
}

func TestBuildMemoryVectorIsNormalized(t *testing.T) {
	vector := BuildMemoryVector("周末一起去杭州旅行", 128)
	score := cosineSimilarity(vector, vector)
	if score < 0.999 || score > 1.001 {
		t.Fatalf("expected normalized self similarity, got %.4f", score)
	}
}
