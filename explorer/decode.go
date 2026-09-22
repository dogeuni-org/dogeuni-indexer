package explorer

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/utils"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/txscript"
)

func (e *Explorer) reDecode(vin btcjson.Vin) (*models.BaseInscription, []byte, error) {

	in := vin
	if in.ScriptSig == nil {
		return nil, nil, errors.New("ScriptSig is nil")
	}

	scriptbytes, err := hex.DecodeString(in.ScriptSig.Hex)
	if err != nil {
		return nil, nil, fmt.Errorf("hex.DecodeString err: %s", err.Error())
	}

	pkScript, err := txscript.PushedData(scriptbytes)
	if err != nil {
		return nil, nil, fmt.Errorf("PushedData err: %s", err.Error())
	}

	if len(pkScript) < 3 {
		return nil, nil, errors.New("pkScript length < 3")
	}

	pushedData, err := txscript.PushedData(pkScript[len(pkScript)-1])
	if err != nil {
		return nil, nil, fmt.Errorf("PushedData err: %s", err.Error())
	}

	if len(pushedData) < 4 {
		return nil, nil, errors.New("len(pushedData) < 4")
	}

	param := &models.BaseInscription{}
	err = json.Unmarshal(pushedData[3], param)
	if err != nil {
		return nil, nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	return param, pushedData[3], nil
}

func (e *Explorer) reDecodeNft(tx *btcjson.TxRawResult) (*models.NftInscription, error) {

	param := &models.NftInscription{}
	imageDatas := []byte{}

	for i, in := range tx.Vin {

		if in.ScriptSig == nil {
			return nil, errors.New("ScriptSig is nil")
		}

		scriptbytes, err := hex.DecodeString(in.ScriptSig.Hex)
		if err != nil {
			return nil, fmt.Errorf("hex.DecodeString err: %s", err.Error())
		}

		pkScript, err := txscript.PushedData(scriptbytes)
		if err != nil {
			return nil, fmt.Errorf("PushedData err: %s", err.Error())
		}

		if len(pkScript) < 3 {
			return nil, errors.New("pkScript length < 3")
		}
		if i == 0 {
			pushedData, err := txscript.PushedData(pkScript[len(pkScript)-1])
			if err != nil {
				return nil, fmt.Errorf("PushedData err: %s", err.Error())
			}

			if len(pushedData) < 4 {
				return nil, errors.New("len(pushedData) < 4")
			}

			param = &models.NftInscription{}
			err = json.Unmarshal(pushedData[3], param)
			if err != nil {
				return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
			}
			continue
		}

		pushedData, err := txscript.PushedData(pkScript[len(pkScript)-1])
		if err != nil {
			return nil, fmt.Errorf("PushedData err: %s", err.Error())
		}

		imageDatas = append(imageDatas, pushedData[len(pushedData)-1]...)

		for _, pk := range pkScript {
			if pk == nil {
				break
			}
			imageDatas = append(imageDatas, pk...)
		}
	}

	param.Image = utils.PngToBase64(imageDatas)

	return param, nil
}

func (e *Explorer) reDecodeFile(tx *btcjson.TxRawResult) (*models.FileInscription, error) {

	inscription := &models.FileInscription{}
	fileDatas := []byte{}

	for i, in := range tx.Vin {

		if in.ScriptSig == nil {
			return nil, errors.New("ScriptSig is nil")
		}

		scriptbytes, err := hex.DecodeString(in.ScriptSig.Hex)
		if err != nil {
			return nil, fmt.Errorf("hex.DecodeString err: %s", err.Error())
		}

		pkScript, err := txscript.PushedData(scriptbytes)
		if err != nil {
			return nil, fmt.Errorf("PushedData err: %s", err.Error())
		}

		if len(pkScript) < 3 {
			return nil, errors.New("pkScript length < 3")
		}
		if i == 0 {
			pushedData, err := txscript.PushedData(pkScript[len(pkScript)-1])
			if err != nil {
				return nil, fmt.Errorf("PushedData err: %s", err.Error())
			}

			if len(pushedData) < 4 {
				return nil, errors.New("len(pushedData) < 4")
			}

			inscription = &models.FileInscription{}
			err = json.Unmarshal(pushedData[3], inscription)
			if err != nil {
				return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
			}
			continue
		}

		pushedData, err := txscript.PushedData(pkScript[len(pkScript)-1])
		if err != nil {
			return nil, fmt.Errorf("pushedData err: %s", err.Error())
		}

		fileDatas = append(fileDatas, pushedData[len(pushedData)-1]...)

		for _, pk := range pkScript {
			if pk == nil {
				break
			}
			fileDatas = append(fileDatas, pk...)
		}
	}

	inscription.File = fileDatas

	return inscription, nil
}

// outputAddress returns the first address of output i of tx. Inscription txs are
// untrusted input: a missing output, or a script without a standard address
// (OP_RETURN, non-standard), makes the inscription invalid rather than crashing the scan.
func outputAddress(tx *btcjson.TxRawResult, i int) (string, error) {
	if i < 0 || i >= len(tx.Vout) {
		return "", fmt.Errorf("tx %s has no output %d", tx.Txid, i)
	}
	addrs := tx.Vout[i].ScriptPubKey.Addresses
	if len(addrs) == 0 {
		return "", fmt.Errorf("tx %s output %d has no address", tx.Txid, i)
	}
	return addrs[0], nil
}
