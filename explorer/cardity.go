package explorer

import (
	"crypto/sha256"
	"dogeuni-indexer/models"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dogecoinw/doged/chaincfg/chainhash"
	"github.com/dogecoinw/go-dogecoin/log"
	"strings"
	"time"
	"unicode/utf8"
)

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func hasCRAC(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	// ASCII "CRAC" (Carc magic reversed or specific marker per convention)
	return (b[0] == 'C' && b[1] == 'R' && b[2] == 'A' && b[3] == 'C')
}

// minimal envelope for p=cardity inscriptions/opreturn
type cardityEnvelope struct {
	P           string          `json:"p"`
	Protocol    string          `json:"protocol"`
	Version     string          `json:"version"`
	Op          string          `json:"op"`
	ContractId  string          `json:"contract_id"`
	Abi         json.RawMessage `json:"abi"` // accept object or string
	AbiCID      string          `json:"abi_cid"`
	CarcB64     string          `json:"carc_b64"`
	CarB64      string          `json:"car"` // compatibility: raw car json/base64 if provided
	Method      string          `json:"method"`
	Args        json.RawMessage `json:"args"`
	ContractRef string          `json:"contract_ref"`
	// package/module support
	PackageId  string          `json:"package_id"`
	ModuleName string          `json:"module"`
	Modules    json.RawMessage `json:"modules"`
	// shard support
	BundleId string `json:"bundle_id"`
	Idx      *int   `json:"idx"`
	Total    *int   `json:"total"`
	// sdk/contentType support
	ContentType string `json:"content_type"`
	FileB64     string `json:"file_b64"`
	FileHex     string `json:"file_hex"`
}

func (e *Explorer) cardityDecode(rawJSON string) (*cardityEnvelope, error) {
	// 4KB guard and UTF-8 validation
	if len(rawJSON) > 4*1024 {
		log.Warn("cardity", "decode", "payload too large", "len", len(rawJSON))
		return nil, fmt.Errorf("payload too large")
	}
	if !utf8.ValidString(rawJSON) {
		log.Warn("cardity", "decode", "invalid utf8")
		return nil, fmt.Errorf("invalid utf8")
	}

	// try direct JSON first
	b := []byte(strings.TrimSpace(rawJSON))
	env := &cardityEnvelope{}
	if err := json.Unmarshal(b, env); err != nil {
		// try hex-wrapped JSON
		hx := strings.TrimPrefix(string(b), "0x")
		if hb, herr := hex.DecodeString(hx); herr == nil {
			if json.Unmarshal(hb, env) == nil {
				b = hb
			}
		}
	}
	// validate protocol marker
	pl := strings.ToLower(env.P)
	if pl != "cardity" && pl != "cardinals" && pl != "cpl" {
		return nil, fmt.Errorf("not cardity payload")
	}
	// alias handling: hex -> FileHex if not set
	var mp map[string]interface{}
	if json.Unmarshal(b, &mp) == nil {
		if env.FileHex == "" {
			if v, ok := mp["hex"].(string); ok && v != "" {
				env.FileHex = v
			}
		}
	}
	return env, nil
}

func (e *Explorer) executeCardity(txhash, blockHash string, height int64, rawJSON string) error {
	env, err := e.cardityDecode(rawJSON)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	// derive from address from vin[0] prevout
	fromAddress := ""
	if h, herr := chainhash.NewHashFromStr(txhash); herr == nil {
		if txv, terr := e.node.GetRawTransactionVerboseBool(h); terr == nil && len(txv.Vin) > 0 {
			vin0 := txv.Vin[0]
			if ph, phErr := chainhash.NewHashFromStr(vin0.Txid); phErr == nil {
				if prev, prevErr := e.node.GetRawTransactionVerboseBool(ph); prevErr == nil {
					voutIdx := vin0.Vout
					if int(voutIdx) < len(prev.Vout) {
						addrs := prev.Vout[voutIdx].ScriptPubKey.Addresses
						if len(addrs) > 0 {
							fromAddress = addrs[0]
						}
					}
				}
			}
		}
	}
	switch strings.ToLower(env.Op) {
	case "deploy":
		// compute a contract id fallback if absent
		contractId := env.ContractId
		if contractId == "" {
			contractId = txhash
		}
		abiText := ""
		if len(env.Abi) > 0 {
			// store as-is textual JSON (object or string)
			abiText = string(env.Abi)
		}
		carcB64 := firstNonEmpty(env.CarcB64, env.FileB64)
		carcHex := env.FileHex
		var sha string
		var size int64
		if carcB64 != "" {
			if b, err := base64.StdEncoding.DecodeString(carcB64); err == nil {
				if hasCRAC(b) {
					sha = fmt.Sprintf("%x", sha256.Sum256(b))
					size = int64(len(b))
				} else {
					log.Warn("cardity", "deploy", "invalid carc magic", "tx", txhash)
				}
			}
		} else if carcHex != "" {
			if b, err := hex.DecodeString(strings.TrimPrefix(carcHex, "0x")); err == nil {
				if hasCRAC(b) {
					sha = fmt.Sprintf("%x", sha256.Sum256(b))
					size = int64(len(b))
				} else {
					log.Warn("cardity", "deploy", "invalid carc magic", "tx", txhash)
				}
			}
		}
		c := &models.CardityContract{
			ContractId:   contractId,
			Protocol:     env.Protocol,
			Version:      env.Version,
			AbiJSON:      abiText,
			AbiCID:       env.AbiCID,
			CarcSHA256:   sha,
			Size:         size,
			PackageId:    env.PackageId,
			ModuleName:   env.ModuleName,
			DeployTxHash: txhash,
			Creator:      fromAddress,
			BlockHash:    blockHash,
			BlockNumber:  height,
			CreateDate:   now,
		}
		if err := e.dbc.SaveCardityContract(c); err != nil {
			return fmt.Errorf("SaveCardityContract err: %v", err)
		}
		return nil
	case "deploy_package":
		pkg := &models.CardityPackage{
			PackageId:    env.PackageId,
			Version:      env.Version,
			PackageABI:   string(env.Abi),
			ModulesJSON:  string(env.Modules),
			DeployTxHash: txhash,
			BlockHash:    blockHash,
			BlockNumber:  height,
			CreateDate:   now,
		}
		if err := e.dbc.SaveCardityPackage(pkg); err != nil {
			return fmt.Errorf("SaveCardityPackage err: %v", err)
		}
		return nil
	case "deploy_part":
		idx, total := 0, 0
		if env.Idx != nil {
			idx = *env.Idx
		}
		if env.Total != nil {
			total = *env.Total
		}
		part := &models.CardityBundlePart{
			BundleId:    env.BundleId,
			Idx:         idx,
			Total:       total,
			PackageId:   env.PackageId,
			Version:     env.Version,
			ModuleName:  env.ModuleName,
			AbiJSON:     string(env.Abi),
			CarcB64:     env.CarcB64,
			TxHash:      txhash,
			BlockHash:   blockHash,
			BlockNumber: height,
			CreateDate:  now,
		}
		if err := e.dbc.SaveBundlePart(part); err != nil {
			return fmt.Errorf("SaveBundlePart err: %v", err)
		}
		if env.BundleId != "" {
			parts, err := e.dbc.FindBundleParts(env.BundleId)
			if err == nil && len(parts) > 0 && parts[0].Total > 0 && len(parts) == parts[0].Total {
				for _, p := range parts {
					var sha string
					var size int64
					if p.CarcB64 != "" {
						if b, err := base64.StdEncoding.DecodeString(p.CarcB64); err == nil {
							if hasCRAC(b) {
								sha = fmt.Sprintf("%x", sha256.Sum256(b))
								size = int64(len(b))
							} else {
								log.Warn("cardity", "deploy_part", "invalid carc magic", "tx", p.TxHash)
							}
						}
					}
					_ = e.dbc.SaveCardityModule(&models.CardityModule{
						PackageId:    p.PackageId,
						Name:         p.ModuleName,
						AbiJSON:      p.AbiJSON,
						CarcB64:      p.CarcB64,
						CarcSHA256:   sha,
						Size:         size,
						DeployTxHash: p.TxHash,
						BlockHash:    p.BlockHash,
						BlockNumber:  p.BlockNumber,
						CreateDate:   p.CreateDate,
					})
				}
				_ = e.dbc.SaveCardityPackage(&models.CardityPackage{PackageId: env.PackageId, Version: env.Version})
			}
		}
		return nil

	case "invoke":
		contractId := env.ContractId
		if contractId == "" {
			contractId = env.ContractRef
		}
		method := env.Method
		if env.ModuleName != "" && !strings.Contains(method, ".") {
			method = env.ModuleName + "." + method
		}
		fqn := method
		inv := &models.CardityInvocationLog{
			ContractId:  contractId,
			Method:      method,
			MethodFQN:   fqn,
			ArgsJSON:    string(env.Args),
			ArgsText:    truncate(string(env.Args), 240),
			FromAddress: fromAddress,
			TxHash:      txhash,
			BlockHash:   blockHash,
			BlockNumber: height,
			CreateDate:  now,
		}
		if err := e.dbc.SaveCardityInvocation(inv); err != nil {
			return fmt.Errorf("SaveCardityInvocation err: %v", err)
		}
		// M1: no runtime execution, events are optional and skipped
		return nil
	default:
		return fmt.Errorf("unsupported cardity op: %s", env.Op)
	}
}

// helper to pull JSON from pushedData if needed later; for M1 we expect raw json string provided by caller

// mirrorCardityDrc20 creates Cardity logs for DRC-20 ops when enabled
func (e *Explorer) mirrorCardityDrc20(d *models.Drc20Info) {
	if e.config == nil || !e.config.Cardity.Enable {
		return
	}
	now := time.Now().Unix()
	contractId := strings.ToUpper(d.Tick)

	// deploy → ensure contract exists
	if strings.ToLower(d.Op) == "deploy" {
		_ = e.dbc.SaveCardityContract(&models.CardityContract{
			ContractId:   contractId,
			Protocol:     "DRC20",
			Version:      "1.0",
			DeployTxHash: d.TxHash,
			BlockHash:    d.BlockHash,
			BlockNumber:  d.BlockNumber,
			CreateDate:   now,
		})
		return
	}

	// mint / transfer
	var (
		method string
		args   []interface{}
	)
	switch strings.ToLower(d.Op) {
	case "mint":
		method = "mint"
		args = []interface{}{d.HolderAddress, d.Amt.Int().String(), contractId}
	case "transfer":
		method = "transfer"
		args = []interface{}{d.HolderAddress, d.ToAddress, d.Amt.Int().String(), contractId}
	default:
		return
	}

	argsb, _ := json.Marshal(args)
	_ = e.dbc.SaveCardityInvocation(&models.CardityInvocationLog{
		ContractId:  contractId,
		Method:      method,
		ArgsJSON:    string(argsb),
		TxHash:      d.TxHash,
		BlockHash:   d.BlockHash,
		BlockNumber: d.BlockNumber,
		CreateDate:  now,
	})
}

// mirrorCardityMeme20 creates Cardity logs for Meme20 ops when enabled
func (e *Explorer) mirrorCardityMeme20(m *models.Meme20Info) {
	if e.config == nil || !e.config.Cardity.Enable {
		return
	}
	now := time.Now().Unix()
	contractId := m.TickId

	if strings.ToLower(m.Op) == "deploy" {
		_ = e.dbc.SaveCardityContract(&models.CardityContract{
			ContractId:   contractId,
			Protocol:     "MEME20",
			Version:      "1.0",
			DeployTxHash: m.TxHash,
			BlockHash:    m.BlockHash,
			BlockNumber:  m.BlockNumber,
			CreateDate:   now,
		})
		return
	}

	if strings.ToLower(m.Op) == "transfer" {
		args, _ := json.Marshal([]interface{}{m.HolderAddress, m.ToAddress, m.Amt.Int().String(), contractId})
		_ = e.dbc.SaveCardityInvocation(&models.CardityInvocationLog{
			ContractId:  contractId,
			Method:      "transfer",
			ArgsJSON:    string(args),
			TxHash:      m.TxHash,
			BlockHash:   m.BlockHash,
			BlockNumber: m.BlockNumber,
			CreateDate:  now,
		})
	}
}
