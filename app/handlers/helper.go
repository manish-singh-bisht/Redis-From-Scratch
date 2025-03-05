package handlers

import RESP "github.com/manish-singh-bisht/Redis-From-Scratch/app/resp"

/*
 	* HandleError handles an error
	* @param writer *RESP.Writer - the writer to write to
	* @param errorMsg []byte - the error message to write
	* @return error - the error if there is one
*/
func HandleError(writer *RESP.Writer, errorMsg []byte) error {
	return writer.Encode(&RESP.RESPMessage{
		Type:  RESP.Error,
		Value: errorMsg,
	})
}
