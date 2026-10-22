package classifier

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"tunbridge/internal/db"
)

type Engine struct {
	Store *db.Store
}

func NewEngine(store *db.Store) *Engine {
	return &Engine{Store: store}
}

type ClassificationResult struct {
	BucketID      int64                `json:"bucket_id"`
	BucketName    string               `json:"bucket_name"`
	BucketLabel   string               `json:"bucket_label"`
	BucketColor   string               `json:"bucket_color"`
	Confidence    float64              `json:"confidence"`
	Probabilities map[string]float64   `json:"probabilities"`
	LogScores     map[string]float64   `json:"log_scores"`
	MagnetMatched *db.Magnet           `json:"magnet_matched,omitempty"`
	TopWords      []WordScoreDetail    `json:"top_words,omitempty"`
	BucketDetails []BucketScoreSummary `json:"bucket_details,omitempty"`
}

type BucketScoreSummary struct {
	BucketID    int64   `json:"bucket_id"`
	BucketName  string  `json:"bucket_name"`
	BucketLabel string  `json:"bucket_label"`
	BucketColor string  `json:"bucket_color"`
	Probability float64 `json:"probability"`
	LogScore    float64 `json:"log_score"`
	MatchWords  int     `json:"match_words"`
}

type WordScoreDetail struct {
	Word           string             `json:"word"`
	DisplayWord    string             `json:"display_word"`
	FrequencyInMsg int                `json:"frequency_in_msg"`
	BestBucket     string             `json:"best_bucket"`
	BestLabel      string             `json:"best_label"`
	BestColor      string             `json:"best_color"`
	MaxProb        float64            `json:"max_prob"`
	ImpactScore    float64            `json:"impact_score"`
	BucketProbs    map[string]float64 `json:"bucket_probs"`
	BucketCounts   map[string]int     `json:"bucket_counts"`
}

type CorpusWordItem struct {
	Word       string         `json:"word"`
	Count      int            `json:"count"`
	Prob       float64        `json:"prob"`
	AllBuckets map[string]int `json:"all_buckets"`
}

type EngineStats struct {
	TotalEmails        int     `json:"total_emails"`
	ClassifiedEmails   int     `json:"classified_emails"`
	ReclassifiedCount  int     `json:"reclassified_count"`
	MagnetCount        int     `json:"magnet_count"`
	MagnetHits         int     `json:"magnet_hits"`
	AccuracyRate       float64 `json:"accuracy_rate"`
	TotalCorpusWords   int     `json:"total_corpus_words"`
	UniqueVocabSize    int     `json:"unique_vocab_size"`
	TotalTrainedEmails int     `json:"total_trained_emails"`
}

// MatchMagnets checks enabled magnets in order and returns the first matching magnet.
func (e *Engine) MatchMagnets(fromName, fromAddr, toAddr, subject string) (*db.Magnet, error) {
	magnets, err := e.Store.GetMagnets()
	if err != nil {
		return nil, err
	}
	fromCombined := strings.ToLower(fromName + " <" + fromAddr + ">")
	toLower := strings.ToLower(toAddr)
	subjLower := strings.ToLower(subject)
	domainLower := ""
	if at := strings.LastIndex(fromAddr, "@"); at != -1 && at+1 < len(fromAddr) {
		domainLower = strings.ToLower(fromAddr[at+1:])
	}

	for _, m := range magnets {
		if !m.Enabled {
			continue
		}
		pat := strings.ToLower(strings.TrimSpace(m.Pattern))
		if pat == "" {
			continue
		}
		matched := false
		switch m.Field {
		case "from":
			matched = strings.Contains(fromCombined, pat)
		case "to":
			matched = strings.Contains(toLower, pat)
		case "subject":
			matched = strings.Contains(subjLower, pat)
		case "domain":
			matched = strings.Contains(domainLower, pat)
		}
		if matched {
			return &m, nil
		}
	}
	return nil, nil
}

// ClassifyAndExplain runs POPFile magnet check + Naive Bayes classification and returns full breakdown.
func (e *Engine) ClassifyAndExplain(fromName, fromAddr, toAddr, subject, bodyText, bodyHTML string, incrementMagnetHit bool) (*ClassificationResult, error) {
	buckets, err := e.Store.GetBuckets()
	if err != nil {
		return nil, err
	}

	bucketMap := make(map[int64]db.Bucket)
	bucketByName := make(map[string]db.Bucket)
	var activeBuckets []db.Bucket
	totalTrainedDocs := 0
	totalCorpusWords := 0

	for _, b := range buckets {
		bucketMap[b.ID] = b
		bucketByName[b.Name] = b
		if b.Name != "unclassified" {
			activeBuckets = append(activeBuckets, b)
			totalTrainedDocs += b.TrainedDocs
			totalCorpusWords += b.TotalWords
		}
	}

	// Tokenize message
	tokens := TokenizeEmail(fromName, fromAddr, toAddr, subject, bodyText, bodyHTML)

	// Check Magnets first
	magnet, _ := e.MatchMagnets(fromName, fromAddr, toAddr, subject)
	if magnet != nil && incrementMagnetHit {
		_, _ = e.Store.DB.Exec("UPDATE magnets SET hit_count = hit_count + 1 WHERE id = ?", magnet.ID)
	}

	// Fetch vocabulary size
	var vocabSize int
	_ = e.Store.DB.QueryRow("SELECT COUNT(DISTINCT word) FROM word_counts WHERE count > 0").Scan(&vocabSize)
	if vocabSize < 100 {
		vocabSize = 100
	}

	// Look up token counts across all buckets in one query
	wordBucketCounts := make(map[string]map[int64]int)
	if len(tokens) > 0 {
		words := make([]string, 0, len(tokens))
		for w := range tokens {
			words = append(words, w)
		}
		// Query in batches of 200
		for i := 0; i < len(words); i += 200 {
			end := i + 200
			if end > len(words) {
				end = len(words)
			}
			chunk := words[i:end]
			placeholders := make([]string, len(chunk))
			args := make([]any, len(chunk))
			for j, w := range chunk {
				placeholders[j] = "?"
				args[j] = w
			}
			q := fmt.Sprintf("SELECT bucket_id, word, count FROM word_counts WHERE count > 0 AND word IN (%s)", strings.Join(placeholders, ","))
			rows, err := e.Store.DB.Query(q, args...)
			if err == nil {
				for rows.Next() {
					var bid int64
					var w string
					var c int
					if err := rows.Scan(&bid, &w, &c); err == nil {
						if wordBucketCounts[w] == nil {
							wordBucketCounts[w] = make(map[int64]int)
						}
						wordBucketCounts[w][bid] = c
					}
				}
				rows.Close()
			}
		}
	}

	// If corpus has zero trained words and no magnet matched, return unclassified
	unclassifiedBucket := bucketByName["unclassified"]
	if totalCorpusWords == 0 && magnet == nil {
		probs := make(map[string]float64)
		for _, b := range activeBuckets {
			probs[b.Name] = 1.0 / float64(len(activeBuckets))
		}
		return &ClassificationResult{
			BucketID:      unclassifiedBucket.ID,
			BucketName:    unclassifiedBucket.Name,
			BucketLabel:   unclassifiedBucket.Label,
			BucketColor:   unclassifiedBucket.Color,
			Confidence:    0.0,
			Probabilities: probs,
			LogScores:     make(map[string]float64),
		}, nil
	}

	// Compute POPFile Naive Bayes log-probabilities
	// Laplace smoothing constant alpha = 0.5
	const alpha = 0.5
	logScores := make(map[string]float64)
	matchCounts := make(map[string]int)

	for _, b := range activeBuckets {
		// Prior probability with smoothing
		prior := (float64(b.TrainedDocs) + 1.0) / (float64(totalTrainedDocs) + float64(len(activeBuckets)))
		logScores[b.Name] = math.Log(prior)
	}

	var wordDetails []WordScoreDetail

	for word, freq := range tokens {
		countsByB := wordBucketCounts[word]
		totalWordInCorpus := 0
		for _, c := range countsByB {
			totalWordInCorpus += c
		}

		// Only words present in the corpus contribute to differentiation (like POPFile)
		if totalWordInCorpus == 0 {
			continue
		}

		bucketProbs := make(map[string]float64)
		bucketCountsStr := make(map[string]int)
		sumLikelihood := 0.0

		for _, b := range activeBuckets {
			c := countsByB[b.ID]
			bucketCountsStr[b.Name] = c
			if c > 0 {
				matchCounts[b.Name]++
			}
			// P(w | B_k)
			likelihood := (float64(c) + alpha) / (float64(b.TotalWords) + alpha*float64(vocabSize))
			bucketProbs[b.Name] = likelihood
			sumLikelihood += likelihood

			// Add to bucket log score (damped frequency log(1+freq) + 1 to prevent length domination)
			effectiveFreq := 1.0 + math.Log(float64(freq))
			logScores[b.Name] += effectiveFreq * math.Log(likelihood)
		}

		// Normalize word probability across buckets for POPFile Word Matrix visualization:
		// P(B_k | w) assuming uniform prior for word inspection
		bestBucket := ""
		bestLabel := ""
		bestColor := ""
		maxProb := 0.0
		minProb := 1.0

		for _, b := range activeBuckets {
			normP := 0.0
			if sumLikelihood > 0 {
				normP = bucketProbs[b.Name] / sumLikelihood
			}
			bucketProbs[b.Name] = normP
			if normP > maxProb {
				maxProb = normP
				bestBucket = b.Name
				bestLabel = b.Label
				bestColor = b.Color
			}
			if normP < minProb {
				minProb = normP
			}
		}

		displayW := word
		impact := (maxProb - minProb) * (1.0 + 0.3*math.Log(float64(freq)))

		wordDetails = append(wordDetails, WordScoreDetail{
			Word:           word,
			DisplayWord:    displayW,
			FrequencyInMsg: freq,
			BestBucket:     bestBucket,
			BestLabel:      bestLabel,
			BestColor:      bestColor,
			MaxProb:        maxProb,
			ImpactScore:    impact,
			BucketProbs:    bucketProbs,
			BucketCounts:   bucketCountsStr,
		})
	}

	// Sort words by ImpactScore descending
	sort.Slice(wordDetails, func(i, j int) bool {
		return wordDetails[i].ImpactScore > wordDetails[j].ImpactScore
	})

	// Convert logScores to normalized posterior probabilities via Log-Sum-Exp
	maxLog := -1e300
	for _, ls := range logScores {
		if ls > maxLog {
			maxLog = ls
		}
	}
	sumExp := 0.0
	probs := make(map[string]float64)
	for name, ls := range logScores {
		// Scale temperature gently if token count is huge so probabilities remain interpretable
		val := math.Exp(ls - maxLog)
		probs[name] = val
		sumExp += val
	}
	for name := range probs {
		if sumExp > 0 {
			probs[name] /= sumExp
		}
	}

	// Determine winning bucket
	var winner db.Bucket
	bestP := -1.0
	for _, b := range activeBuckets {
		if probs[b.Name] > bestP {
			bestP = probs[b.Name]
			winner = b
		}
	}

	// If magnet matched, override winner with magnet's bucket!
	if magnet != nil {
		if mb, ok := bucketMap[magnet.BucketID]; ok {
			winner = mb
			bestP = 1.0
			probs[winner.Name] = 1.0
			for _, b := range activeBuckets {
				if b.ID != winner.ID {
					probs[b.Name] = 0.0
				}
			}
		}
	} else if len(wordDetails) == 0 || bestP < 0.32 {
		// Below POPFile unclassified threshold
		winner = unclassifiedBucket
	}

	var details []BucketScoreSummary
	for _, b := range activeBuckets {
		details = append(details, BucketScoreSummary{
			BucketID:    b.ID,
			BucketName:  b.Name,
			BucketLabel: b.Label,
			BucketColor: b.Color,
			Probability: probs[b.Name],
			LogScore:    logScores[b.Name],
			MatchWords:  matchCounts[b.Name],
		})
	}
	sort.Slice(details, func(i, j int) bool {
		return details[i].Probability > details[j].Probability
	})

	if len(wordDetails) > 80 {
		wordDetails = wordDetails[:80]
	}

	return &ClassificationResult{
		BucketID:      winner.ID,
		BucketName:    winner.Name,
		BucketLabel:   winner.Label,
		BucketColor:   winner.Color,
		Confidence:    bestP,
		Probabilities: probs,
		LogScores:     logScores,
		MagnetMatched: magnet,
		TopWords:      wordDetails,
		BucketDetails: details,
	}, nil
}

// TrainEmail adds (or subtracts if sign < 0) an email's token frequencies to a bucket's corpus.
func (e *Engine) TrainEmail(bucketID int64, fromName, fromAddr, toAddr, subject, bodyText, bodyHTML string, sign int) error {
	tokens := TokenizeEmail(fromName, fromAddr, toAddr, subject, bodyText, bodyHTML)
	if len(tokens) == 0 {
		return nil
	}

	tx, err := e.Store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	totalDelta := 0
	for word, count := range tokens {
		delta := count * sign
		totalDelta += delta
		if sign > 0 {
			_, err := tx.Exec(`
				INSERT INTO word_counts (bucket_id, word, count) VALUES (?, ?, ?)
				ON CONFLICT(bucket_id, word) DO UPDATE SET count = count + excluded.count
			`, bucketID, word, count)
			if err != nil {
				return err
			}
		} else {
			_, err := tx.Exec(`
				UPDATE word_counts SET count = MAX(0, count - ?) WHERE bucket_id = ? AND word = ?
			`, count, bucketID, word)
			if err != nil {
				return err
			}
		}
	}

	// Clean up zero counts
	if sign < 0 {
		_, _ = tx.Exec("DELETE FROM word_counts WHERE bucket_id = ? AND count <= 0", bucketID)
	}

	docDelta := 1
	if sign < 0 {
		docDelta = -1
	}
	_, err = tx.Exec(`
		UPDATE buckets 
		SET trained_docs = MAX(0, trained_docs + ?),
		    total_words = MAX(0, total_words + ?)
		WHERE id = ?
	`, docDelta, totalDelta, bucketID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ReclassifyEmail performs POPFile TOE (Train On Error):
// If the email was previously trained in oldBucketID, removes its words from oldBucketID,
// trains its words into newBucketID, updates accuracy counters (false_pos / false_neg),
// and re-evaluates email probabilities.
func (e *Engine) ReclassifyEmail(emailID int64, newBucketID int64) (*db.Email, error) {
	em, err := e.Store.GetEmail(emailID)
	if err != nil {
		return nil, err
	}
	newBucket, err := e.Store.GetBucketByID(newBucketID)
	if err != nil {
		return nil, err
	}
	oldBucket, _ := e.Store.GetBucketByID(em.BucketID)

	// If already trained in previous bucket and bucket is changing, untrain old bucket
	if em.IsTrained && em.BucketID != newBucketID && em.BucketName != "unclassified" {
		if err := e.TrainEmail(em.BucketID, em.FromName, em.FromAddress, em.ToAddress, em.Subject, em.BodyText, em.BodyHTML, -1); err != nil {
			return nil, err
		}
	}

	// Train in new bucket (if not unclassified)
	if newBucket.Name != "unclassified" {
		if err := e.TrainEmail(newBucketID, em.FromName, em.FromAddress, em.ToAddress, em.Subject, em.BodyText, em.BodyHTML, 1); err != nil {
			return nil, err
		}
	}

	// Record POPFile False Positive / False Negative metrics if changing from original prediction
	if em.BucketID != newBucketID {
		_, _ = e.Store.DB.Exec("UPDATE buckets SET false_pos = false_pos + 1 WHERE id = ?", em.BucketID)
		_, _ = e.Store.DB.Exec("UPDATE buckets SET false_neg = false_neg + 1 WHERE id = ?", newBucketID)
	}

	// Re-compute scores after training
	res, err := e.ClassifyAndExplain(em.FromName, em.FromAddress, em.ToAddress, em.Subject, em.BodyText, em.BodyHTML, false)
	scoresJSON := em.ScoresJSON
	conf := 1.0
	if err == nil && res != nil {
		if b, err := json.Marshal(res.Probabilities); err == nil {
			scoresJSON = string(b)
		}
		if p, ok := res.Probabilities[newBucket.Name]; ok {
			conf = p
		}
	}

	// Update X-Text-Classification header in RawHeaders
	updatedHeaders := InjectPOPFileHeader(em.RawHeaders, newBucket.Name)

	isReclass := 0
	if em.OriginalBucketID != newBucketID {
		isReclass = 1
	}

	_, err = e.Store.DB.Exec(`
		UPDATE emails 
		SET bucket_id = ?, confidence = ?, scores_json = ?, is_trained = 1, is_reclassified = ?, raw_headers = ?
		WHERE id = ?
	`, newBucketID, conf, scoresJSON, isReclass, updatedHeaders, emailID)
	if err != nil {
		return nil, err
	}

	oldName := ""
	if oldBucket != nil {
		oldName = oldBucket.Label
	}
	e.Store.AddHistoryEvent(em.ID, em.Subject, em.FromAddress, "reclassified", oldName, newBucket.Label)

	return e.Store.GetEmail(emailID)
}

// ReclassifyAllUnclassified re-runs the classifier on all emails (or unclassified emails) using current magnets & corpus.
func (e *Engine) RescanEmails(onlyUnclassified bool) (int, error) {
	filter := db.EmailFilter{Folder: "received", Limit: 500}
	if onlyUnclassified {
		if unc, err := e.Store.GetBucketByName("unclassified"); err == nil {
			filter.BucketID = unc.ID
		}
	}
	emails, err := e.Store.ListEmails(filter)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, em := range emails {
		if em.IsReclassified && !onlyUnclassified {
			continue
		}
		res, err := e.ClassifyAndExplain(em.FromName, em.FromAddress, em.ToAddress, em.Subject, em.BodyText, em.BodyHTML, false)
		if err != nil || res == nil {
			continue
		}
		scoresB, _ := json.Marshal(res.Probabilities)
		magID := int64(0)
		if res.MagnetMatched != nil {
			magID = res.MagnetMatched.ID
		}
		headers := InjectPOPFileHeader(em.RawHeaders, res.BucketName)
		_, err = e.Store.DB.Exec(`
			UPDATE emails SET bucket_id = ?, magnet_id = ?, confidence = ?, scores_json = ?, raw_headers = ?
			WHERE id = ?
		`, res.BucketID, magID, res.Confidence, string(scoresB), headers, em.ID)
		if err == nil {
			updated++
		}
	}
	return updated, nil
}

// InjectPOPFileHeader adds or updates X-Text-Classification header (classic POPFile header).
func InjectPOPFileHeader(rawHeaders, bucketName string) string {
	lines := strings.Split(rawHeaders, "\n")
	var out []string
	found := false
	for _, l := range lines {
		trimmed := strings.TrimRight(l, "\r")
		if strings.HasPrefix(strings.ToLower(trimmed), "x-text-classification:") {
			out = append(out, "X-Text-Classification: "+bucketName)
			found = true
		} else if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if !found {
		out = append(out, "X-Text-Classification: "+bucketName)
	}
	return strings.Join(out, "\n")
}

// GetCorpusWords returns top words for a specific bucket with cross-bucket counts.
func (e *Engine) GetCorpusWords(bucketID int64, search string, limit int) ([]CorpusWordItem, error) {
	if limit <= 0 {
		limit = 100
	}
	b, err := e.Store.GetBucketByID(bucketID)
	if err != nil {
		return nil, err
	}

	var rowsQuery string
	var args []any
	if strings.TrimSpace(search) != "" {
		rowsQuery = "SELECT word, count FROM word_counts WHERE bucket_id = ? AND count > 0 AND word LIKE ? ORDER BY count DESC LIMIT ?"
		args = []any{bucketID, "%" + strings.ToLower(strings.TrimSpace(search)) + "%", limit}
	} else {
		rowsQuery = "SELECT word, count FROM word_counts WHERE bucket_id = ? AND count > 0 ORDER BY count DESC LIMIT ?"
		args = []any{bucketID, limit}
	}

	rows, err := e.Store.DB.Query(rowsQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CorpusWordItem
	var wordList []string
	for rows.Next() {
		var it CorpusWordItem
		if err := rows.Scan(&it.Word, &it.Count); err != nil {
			return nil, err
		}
		if b.TotalWords > 0 {
			it.Prob = float64(it.Count) / float64(b.TotalWords)
		}
		it.AllBuckets = make(map[string]int)
		items = append(items, it)
		wordList = append(wordList, it.Word)
	}

	if len(wordList) > 0 {
		placeholders := make([]string, len(wordList))
		qArgs := make([]any, len(wordList))
		for i, w := range wordList {
			placeholders[i] = "?"
			qArgs[i] = w
		}
		q := fmt.Sprintf(`
			SELECT wc.word, bk.name, wc.count
			FROM word_counts wc
			JOIN buckets bk ON wc.bucket_id = bk.id
			WHERE wc.count > 0 AND wc.word IN (%s)
		`, strings.Join(placeholders, ","))
		cRows, err := e.Store.DB.Query(q, qArgs...)
		if err == nil {
			cross := make(map[string]map[string]int)
			for cRows.Next() {
				var w, bname string
				var c int
				if err := cRows.Scan(&w, &bname, &c); err == nil {
					if cross[w] == nil {
						cross[w] = make(map[string]int)
					}
					cross[w][bname] = c
				}
			}
			cRows.Close()
			for i := range items {
				if m, ok := cross[items[i].Word]; ok {
					items[i].AllBuckets = m
				}
			}
		}
	}

	return items, nil
}

// LookupWord looks up a specific token across all buckets in the POPFile corpus.
func (e *Engine) LookupWord(word string) (map[string]any, error) {
	w := strings.ToLower(strings.TrimSpace(word))
	buckets, err := e.Store.GetBuckets()
	if err != nil {
		return nil, err
	}
	rows, err := e.Store.DB.Query("SELECT bucket_id, count FROM word_counts WHERE word = ? AND count > 0", w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[int64]int)
	for rows.Next() {
		var bid int64
		var c int
		if err := rows.Scan(&bid, &c); err == nil {
			counts[bid] = c
		}
	}

	var breakdown []map[string]any
	sumL := 0.0
	likes := make(map[int64]float64)
	for _, b := range buckets {
		if b.Name == "unclassified" {
			continue
		}
		c := counts[b.ID]
		l := 0.0
		if b.TotalWords > 0 && c > 0 {
			l = float64(c) / float64(b.TotalWords)
		}
		likes[b.ID] = l
		sumL += l
	}

	for _, b := range buckets {
		if b.Name == "unclassified" {
			continue
		}
		c := counts[b.ID]
		affinity := 0.0
		if sumL > 0 {
			affinity = likes[b.ID] / sumL
		}
		breakdown = append(breakdown, map[string]any{
			"bucket_id":    b.ID,
			"bucket_name":  b.Name,
			"bucket_label": b.Label,
			"bucket_color": b.Color,
			"count":        c,
			"affinity":     affinity,
		})
	}

	return map[string]any{
		"word":      w,
		"breakdown": breakdown,
	}, nil
}

func (e *Engine) GetStats() (*EngineStats, error) {
	var st EngineStats
	_ = e.Store.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE folder = 'received'").Scan(&st.TotalEmails)
	_ = e.Store.DB.QueryRow("SELECT COUNT(*) FROM emails e JOIN buckets b ON e.bucket_id = b.id WHERE e.folder = 'received' AND b.name != 'unclassified'").Scan(&st.ClassifiedEmails)
	_ = e.Store.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE folder = 'received' AND is_reclassified = 1").Scan(&st.ReclassifiedCount)
	_ = e.Store.DB.QueryRow("SELECT COUNT(*) FROM magnets WHERE enabled = 1").Scan(&st.MagnetCount)
	_ = e.Store.DB.QueryRow("SELECT COALESCE(SUM(hit_count), 0) FROM magnets").Scan(&st.MagnetHits)
	_ = e.Store.DB.QueryRow("SELECT COALESCE(SUM(total_words), 0), COALESCE(SUM(trained_docs), 0) FROM buckets").Scan(&st.TotalCorpusWords, &st.TotalTrainedEmails)
	_ = e.Store.DB.QueryRow("SELECT COUNT(DISTINCT word) FROM word_counts WHERE count > 0").Scan(&st.UniqueVocabSize)

	if st.TotalEmails > 0 {
		correct := st.TotalEmails - st.ReclassifiedCount
		if correct < 0 {
			correct = 0
		}
		st.AccuracyRate = (float64(correct) / float64(st.TotalEmails)) * 100.0
	} else {
		st.AccuracyRate = 100.0
	}
	return &st, nil
}
