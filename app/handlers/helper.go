package handlers

import RESP "github.com/manish-singh-bisht/Redis-From-Scratch/app/resp"

func HandleError(writer *RESP.Writer, errorMsg []byte) error {
	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.Error,
		Value: errorMsg,
	})
}
