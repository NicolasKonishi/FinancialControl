package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/analysis"
	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
	"github.com/NicolasKonishi/FinancialControl/internal/statement"
)

const statementUploadLimit = 8 << 20

// StatementParser extracts purchases from a statement PDF, CSV, or OFX.
type StatementParser interface {
	ParseStatement(ctx context.Context, file []byte, year, month int) (models.ParsedStatement, error)
}

// StatementWallets loads and updates wallets during statement import.
type StatementWallets interface {
	GetWalletByID(ctx context.Context, id int) (models.Wallet, error)
	ListWallets(ctx context.Context) ([]models.Wallet, error)
	UpdateWallet(ctx context.Context, id int, input models.UpdateWalletInput) (models.Wallet, error)
	ReconcileCardInvoice(ctx context.Context, walletID, year, month int, amount float64, periodStart, periodEnd *string) (models.CardInvoice, error)
}

// Statements handles statement preview/import endpoints.
type Statements struct {
	Parser       StatementParser
	Transactions TransactionStore
	Categories   CategoryStore
	Members      MemberStore
	Wallets      StatementWallets
}

// Preview handles POST /statements/preview.
func (h *Statements) Preview(w http.ResponseWriter, r *http.Request) {
	file, year, month, walletID, memberID, ok := h.readUpload(w, r)
	if !ok {
		return
	}

	parsed, ok := h.parseFile(w, r, file, year, month)
	if !ok {
		return
	}
	var destination *models.Wallet
	if parsed.StatementType == "credit_card" {
		destination, walletID, memberID, ok = h.resolveCreditStatementWallet(w, r, parsed.Issuer, walletID, memberID)
		if !ok {
			return
		}
	}

	existing, err := h.Transactions.ListTransactions(r.Context())
	if err != nil {
		writeStoreError(w, err, "transaction not found")
		return
	}
	categories, err := h.Categories.ListCategories(r.Context())
	if err != nil {
		writeStoreError(w, err, "category not found")
		return
	}

	preview := statement.BuildPreview(parsed, existing, categories, walletID, memberID)
	if destination != nil {
		h.applyInvoicePreview(&preview, *destination)
	}
	if len(preview.Items) == 0 {
		http.Error(w, "Não achei movimentações neste extrato. Tente o CSV ou o OFX da Nubank.", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

// Import handles POST /statements/import.
func (h *Statements) Import(w http.ResponseWriter, r *http.Request) {
	var input models.ImportStatementInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(input.Items) == 0 {
		http.Error(w, "Selecione pelo menos um lançamento.", http.StatusBadRequest)
		return
	}
	if len(input.Items) > 500 {
		http.Error(w, "Muitos itens de uma vez.", http.StatusBadRequest)
		return
	}

	wallet, memberID, walletID, ok := h.resolveImportAccounts(w, r, input.WalletID, input.MemberID)
	if !ok {
		return
	}

	previousInvoice := 0.0
	restoreInvoice := false
	if wallet != nil && models.IsCredit(wallet.Kind) && !input.ApplyToInvoice && input.StatementType != "credit_card" {
		previousInvoice = wallet.InvoiceBalance
		restoreInvoice = true
	}
	if input.StatementType == "credit_card" {
		if wallet == nil || !models.IsCredit(wallet.Kind) {
			http.Error(w, "Escolha o cartão referente a esta fatura.", http.StatusBadRequest)
			return
		}
		if input.InvoiceYear == nil || input.InvoiceMonth == nil || *input.InvoiceYear < 2000 || *input.InvoiceMonth < 1 || *input.InvoiceMonth > 12 {
			http.Error(w, "Competência da fatura inválida.", http.StatusBadRequest)
			return
		}
	}

	created := make([]models.Transaction, 0, len(input.Items))
	for _, item := range input.Items {
		txType := strings.ToLower(strings.TrimSpace(item.Type))
		if txType == "" {
			txType = models.TransactionTypeExpense
		}
		tx, ok := h.buildTx(w, r, item.CategoryID, memberID, walletID, txType, item.Description, item.Amount, item.Date)
		if !ok {
			return
		}
		saved, err := h.Transactions.CreateTransaction(r.Context(), tx)
		if err != nil {
			if errors.Is(err, repository.ErrWalletOwner) {
				http.Error(w, "wallet does not belong to this person", http.StatusBadRequest)
				return
			}
			writeStoreError(w, err, "transaction not found")
			return
		}
		created = append(created, saved)
	}

	if restoreInvoice && wallet != nil {
		if _, err := h.Wallets.UpdateWallet(r.Context(), wallet.ID, walletUpdateKeepingInvoice(*wallet, previousInvoice)); err != nil {
			writeStoreError(w, err, "wallet not found")
			return
		}
	}
	if wallet != nil && input.StatementType == "credit_card" && input.StatementBalance != nil {
		if _, err := h.Wallets.ReconcileCardInvoice(
			r.Context(),
			wallet.ID,
			*input.InvoiceYear,
			*input.InvoiceMonth,
			*input.StatementBalance,
			input.PeriodStart,
			input.PeriodEnd,
		); err != nil {
			writeStoreError(w, err, "invoice not found")
			return
		}
	}

	writeJSON(w, http.StatusCreated, models.ImportStatementResult{
		Created: len(created),
		Items:   created,
	})
}

func (h *Statements) resolveCreditStatementWallet(
	w http.ResponseWriter,
	r *http.Request,
	issuer string,
	selectedWalletID, memberID *int,
) (*models.Wallet, *int, *int, bool) {
	if selectedWalletID != nil {
		selected, err := h.Wallets.GetWalletByID(r.Context(), *selectedWalletID)
		if err != nil {
			writeStoreError(w, err, "wallet not found")
			return nil, nil, nil, false
		}
		if models.IsCredit(selected.Kind) {
			id := selected.ID
			if memberID == nil {
				memberID = selected.MemberID
			}
			return &selected, &id, memberID, true
		}
	}

	wallets, err := h.Wallets.ListWallets(r.Context())
	if err != nil {
		writeStoreError(w, err, "wallet not found")
		return nil, nil, nil, false
	}
	issuer = strings.ToLower(strings.TrimSpace(issuer))
	var fallback *models.Wallet
	for i := range wallets {
		candidate := wallets[i]
		if !models.IsCredit(candidate.Kind) {
			continue
		}
		if memberID != nil && candidate.MemberID != nil && *candidate.MemberID != *memberID {
			continue
		}
		if fallback == nil {
			copy := candidate
			fallback = &copy
		}
		if issuer != "" && issuer != "unknown" && strings.Contains(strings.ToLower(candidate.Name), issuer) {
			copy := candidate
			fallback = &copy
			break
		}
	}
	if fallback == nil {
		http.Error(w, "Cadastre o cartão antes de importar a fatura.", http.StatusBadRequest)
		return nil, nil, nil, false
	}
	id := fallback.ID
	if memberID == nil {
		memberID = fallback.MemberID
	}
	return fallback, &id, memberID, true
}

func (h *Statements) applyInvoicePreview(preview *models.StatementPreview, wallet models.Wallet) {
	var anchor time.Time
	if preview.PeriodEnd != nil {
		anchor, _ = time.Parse("2006-01-02", *preview.PeriodEnd)
	}
	if anchor.IsZero() && len(preview.Items) > 0 {
		anchor, _ = time.Parse("2006-01-02", preview.Items[len(preview.Items)-1].Date)
	}
	if anchor.IsZero() {
		anchor = time.Now().UTC()
	}
	// OFX DTEND is the exclusive closing boundary. A purchase immediately
	// before it belongs to the invoice represented by the file.
	cycle := models.CardCycleForPurchase(wallet, anchor.AddDate(0, 0, -1))
	closing := anchor.Format("2006-01-02")
	due := cycle.DueDate.Format("2006-01-02")
	preview.InvoiceYear = &cycle.Year
	preview.InvoiceMonth = &cycle.Month
	preview.ClosingDate = &closing
	preview.DueDate = &due
}

func (h *Statements) readUpload(w http.ResponseWriter, r *http.Request) ([]byte, int, int, *int, *int, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, statementUploadLimit)
	if err := r.ParseMultipartForm(statementUploadLimit); err != nil {
		http.Error(w, "Envie um arquivo de até 8 MB.", http.StatusBadRequest)
		return nil, 0, 0, nil, nil, false
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Envie o extrato (PDF, CSV ou OFX).", http.StatusBadRequest)
		return nil, 0, 0, nil, nil, false
	}
	defer file.Close()

	name := strings.ToLower(header.Filename)
	if name != "" && !strings.HasSuffix(name, ".pdf") && !strings.HasSuffix(name, ".csv") && !strings.HasSuffix(name, ".ofx") && !strings.HasSuffix(name, ".qfx") {
		http.Error(w, "Envie um PDF, CSV ou OFX do extrato.", http.StatusBadRequest)
		return nil, 0, 0, nil, nil, false
	}

	data, err := io.ReadAll(io.LimitReader(file, statementUploadLimit+1))
	if err != nil || len(data) == 0 {
		http.Error(w, "Não consegui ler o arquivo.", http.StatusBadRequest)
		return nil, 0, 0, nil, nil, false
	}
	if len(data) > statementUploadLimit {
		http.Error(w, "Envie um arquivo de até 8 MB.", http.StatusBadRequest)
		return nil, 0, 0, nil, nil, false
	}

	now := time.Now()
	year := formInt(r, "year", now.Year())
	month := formInt(r, "month", int(now.Month()))
	if month < 1 || month > 12 {
		http.Error(w, "invalid month", http.StatusBadRequest)
		return nil, 0, 0, nil, nil, false
	}

	walletID, ok := optionalPositiveID(w, r.FormValue("wallet_id"), "wallet")
	if !ok {
		return nil, 0, 0, nil, nil, false
	}
	memberID, ok := optionalPositiveID(w, r.FormValue("member_id"), "member")
	if !ok {
		return nil, 0, 0, nil, nil, false
	}

	if walletID != nil {
		wallet, err := h.Wallets.GetWalletByID(r.Context(), *walletID)
		if err != nil {
			writeStoreError(w, err, "wallet not found")
			return nil, 0, 0, nil, nil, false
		}
		if memberID != nil && wallet.MemberID != nil && *wallet.MemberID != *memberID && !models.IsCompanyWallet(wallet.Kind) {
			http.Error(w, "wallet does not belong to this person", http.StatusBadRequest)
			return nil, 0, 0, nil, nil, false
		}
		if memberID == nil {
			memberID = wallet.MemberID
		}
	}
	if memberID != nil {
		if _, err := h.Members.GetMemberByID(r.Context(), *memberID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "member not found", http.StatusBadRequest)
				return nil, 0, 0, nil, nil, false
			}
			writeStoreError(w, err, "member not found")
			return nil, 0, 0, nil, nil, false
		}
	}

	return data, year, month, walletID, memberID, true
}

func (h *Statements) parseFile(w http.ResponseWriter, r *http.Request, file []byte, year, month int) (models.ParsedStatement, bool) {
	parsed, localErr := statement.ParseFile(file, year, month)
	if localErr == nil && len(parsed.Items) > 0 {
		return parsed, true
	}
	if h.Parser != nil {
		py, err := h.Parser.ParseStatement(r.Context(), file, year, month)
		if err == nil {
			return py, true
		}
		if localErr == nil {
			return parsed, true
		}
		var parseErr *analysis.StatementParseError
		if errors.As(err, &parseErr) {
			http.Error(w, statementUserMessage(parseErr.Code), http.StatusBadRequest)
			return models.ParsedStatement{}, false
		}
	}
	if localErr != nil {
		http.Error(w, statementUserMessage(localErr.Error()), http.StatusBadRequest)
		return models.ParsedStatement{}, false
	}
	return parsed, true
}

func statementUserMessage(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "encrypted":
		return "Este PDF está protegido por senha. Abra no banco e exporte de novo sem senha."
	case "no_text", "empty", "invalid pdf":
		return "Não achei texto neste PDF. Use o extrato digital, não uma foto."
	default:
		if strings.Contains(strings.ToLower(code), "encrypted") {
			return "Este PDF está protegido por senha. Abra no banco e exporte de novo sem senha."
		}
		if strings.Contains(strings.ToLower(code), "no_text") {
			return "Não achei texto neste PDF. Use o extrato digital, não uma foto."
		}
		return "Não consegui ler este extrato. Tente o CSV ou o OFX, que a Nubank exporta junto."
	}
}

func (h *Statements) resolveImportAccounts(
	w http.ResponseWriter,
	r *http.Request,
	rawWallet, rawMember *int,
) (*models.Wallet, *int, *int, bool) {
	var wallet *models.Wallet
	var walletID *int
	var memberID *int
	if rawWallet != nil && *rawWallet > 0 {
		found, err := h.Wallets.GetWalletByID(r.Context(), *rawWallet)
		if err != nil {
			writeStoreError(w, err, "wallet not found")
			return nil, nil, nil, false
		}
		wallet = &found
		walletID = &found.ID
		if rawMember == nil || *rawMember < 1 {
			memberID = found.MemberID
		}
	}
	if rawMember != nil && *rawMember > 0 {
		if _, err := h.Members.GetMemberByID(r.Context(), *rawMember); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "member not found", http.StatusBadRequest)
				return nil, nil, nil, false
			}
			writeStoreError(w, err, "member not found")
			return nil, nil, nil, false
		}
		memberID = rawMember
	}
	if wallet != nil && memberID != nil && wallet.MemberID != nil && *wallet.MemberID != *memberID && !models.IsCompanyWallet(wallet.Kind) {
		http.Error(w, "wallet does not belong to this person", http.StatusBadRequest)
		return nil, nil, nil, false
	}
	return wallet, memberID, walletID, true
}

func (h *Statements) buildTx(
	w http.ResponseWriter,
	r *http.Request,
	categoryID int,
	memberID *int,
	walletID *int,
	txType, description string,
	amount float64,
	dateStr string,
) (models.Transaction, bool) {
	helper := &Transactions{Store: h.Transactions, Categories: h.Categories, Members: h.Members}
	return helper.buildFromInput(w, r, categoryID, memberID, walletID, txType, description, amount, dateStr)
}

func formInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.FormValue(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func optionalPositiveID(w http.ResponseWriter, raw, label string) (*int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return nil, true
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		http.Error(w, "invalid "+label, http.StatusBadRequest)
		return nil, false
	}
	return &id, true
}

func walletUpdateKeepingInvoice(wallet models.Wallet, invoice float64) models.UpdateWalletInput {
	return models.UpdateWalletInput{
		Name:           wallet.Name,
		Kind:           wallet.Kind,
		MemberID:       wallet.MemberID,
		Balance:        wallet.Balance,
		ClosingDay:     wallet.ClosingDay,
		DueDay:         wallet.DueDay,
		CreditLimit:    wallet.CreditLimit,
		InvoiceBalance: invoice,
	}
}
