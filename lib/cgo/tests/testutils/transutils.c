
#include <stdio.h>
#include <string.h>

#include <criterion/criterion.h>
#include <criterion/new/assert.h>

#include "libskycoin.h"
#include "skyerrors.h"
#include "skystring.h"
#include "skytest.h"
#include "transutil.h"

int makeKeysAndAddress(cipher__PubKey* ppubkey, cipher__SecKey* pseckey, cipher__Address* paddress){
  int result;
  result = SKY_cipher_GenerateKeyPair(ppubkey, pseckey);
  cr_assert(result == SKY_OK, "SKY_cipher_GenerateKeyPair failed");
  result = SKY_cipher_AddressFromPubKey( ppubkey, paddress );
  cr_assert(result == SKY_OK, "SKY_cipher_AddressFromPubKey failed");
  return result;
}

int makeUxBodyWithSecret(coin__UxBody* puxBody, cipher__SecKey* pseckey){
  cipher__PubKey pubkey;
  cipher__Address address;
  int result;

  memset( puxBody, 0, sizeof(coin__UxBody) );
  puxBody->Coins = 1000000;
  puxBody->Hours = 100;

  result = SKY_cipher_GenerateKeyPair(&pubkey, pseckey);
  cr_assert(result == SKY_OK, "SKY_cipher_GenerateKeyPair failed");

  GoSlice slice;
  memset(&slice, 0, sizeof(GoSlice));
  cipher__SHA256 hash;

  result = SKY_cipher_RandByte( 128, (coin__UxArray*)&slice );
  registerMemCleanup( slice.data );
  cr_assert(result == SKY_OK, "SKY_cipher_RandByte failed");
  result = SKY_cipher_SumSHA256( slice, &puxBody->SrcTransaction );
  cr_assert(result == SKY_OK, "SKY_cipher_SumSHA256 failed");

  result = SKY_cipher_AddressFromPubKey( &pubkey, &puxBody->Address );
  cr_assert(result == SKY_OK, "SKY_cipher_AddressFromPubKey failed");
  return result;
}

int makeUxOutWithSecret(coin__UxOut* puxOut, cipher__SecKey* pseckey){
  int result;
  memset( puxOut, 0, sizeof(coin__UxOut) );
  result = makeUxBodyWithSecret(&puxOut->Body, pseckey);
  puxOut->Head.Time = 100;
  puxOut->Head.BkSeq = 2;
  return result;
}

int makeUxBody(coin__UxBody* puxBody){
  cipher__SecKey seckey;
  return makeUxBodyWithSecret(puxBody, &seckey);
}

int makeUxOut(coin__UxOut* puxOut){
  cipher__SecKey seckey;
  return makeUxOutWithSecret(puxOut, &seckey);
}

int makeAddress(cipher__Address* paddress){
  cipher__PubKey pubkey;
  cipher__SecKey seckey;
  cipher__Address address;
  int result;

  result = SKY_cipher_GenerateKeyPair(&pubkey, &seckey);
  cr_assert(result == SKY_OK, "SKY_cipher_GenerateKeyPair failed");

  result = SKY_cipher_AddressFromPubKey( &pubkey, paddress );
  cr_assert(result == SKY_OK, "SKY_cipher_AddressFromPubKey failed");
  return result;
}

coin__Transaction* makeTransactionFromUxOut(coin__UxOut* puxOut, cipher__SecKey* pseckey ){
  int result;
  coin__Transaction* ptransaction = NULL;
  Transaction__Handle handle;
  result  = SKY_coin_Create_Transaction(&handle);
  cr_assert(result == SKY_OK, "SKY_coin_Create_Transaction failed");
  registerHandleClose(handle);
  result = SKY_coin_Get_Transaction_Object( handle, &ptransaction );
  cr_assert(result == SKY_OK, "SKY_coin_Get_Transaction_Object failed");
  cipher__SHA256 sha256;
  result = SKY_coin_UxOut_Hash(puxOut, &sha256);
  cr_assert(result == SKY_OK, "SKY_coin_UxOut_Hash failed");
  GoUint16 r;
  result = SKY_coin_Transaction_PushInput(ptransaction, &sha256, &r);
  cr_assert(result == SKY_OK, "SKY_coin_Transaction_PushInput failed");

  cipher__Address address1, address2;
  result = makeAddress(&address1);
  cr_assert(result == SKY_OK, "makeAddress failed");
  result = makeAddress(&address2);
  cr_assert(result == SKY_OK, "makeAddress failed");

  result = SKY_coin_Transaction_PushOutput(ptransaction, &address1, 1000000, 50);
  cr_assert(result == SKY_OK, "SKY_coin_Transaction_PushOutput failed");
  result = SKY_coin_Transaction_PushOutput(ptransaction, &address2, 5000000, 50);
  cr_assert(result == SKY_OK, "SKY_coin_Transaction_PushOutput failed");

  GoSlice secKeys = { pseckey, 1, 1 };
  result = SKY_coin_Transaction_SignInputs( ptransaction, secKeys );
  cr_assert(result == SKY_OK, "SKY_coin_Transaction_SignInputs failed");
  result = SKY_coin_Transaction_UpdateHeader( ptransaction );
  cr_assert(result == SKY_OK, "SKY_coin_Transaction_UpdateHeader failed");
  return ptransaction;
}

coin__Transaction* makeTransaction(){
  int result;
  coin__UxOut uxOut;
  cipher__SecKey seckey;

  coin__Transaction* ptransaction = NULL;

  result = makeUxOutWithSecret( &uxOut, &seckey );
  cr_assert(result == SKY_OK, "makeUxOutWithSecret failed");
  ptransaction = makeTransactionFromUxOut( &uxOut, &seckey );
  cr_assert(result == SKY_OK, "makeTransactionFromUxOut failed");
  return ptransaction;
}

coin__Transaction* makeEmptyTransaction(){
  int result;
  coin__Transaction* ptransaction = NULL;
  Transaction__Handle handle;
  result  = SKY_coin_Create_Transaction(&handle);
  cr_assert(result == SKY_OK, "SKY_coin_Create_Transaction failed");
  registerHandleClose(handle);
  result = SKY_coin_Get_Transaction_Object( handle, &ptransaction );
  cr_assert(result == SKY_OK, "SKY_coin_Get_Transaction_Object failed");
  return ptransaction;
}


int makeTransactions(GoSlice* transactions, int n){
  void * data = malloc(sizeof(coin__Transaction) * n);
  if(data == NULL)
    return SKY_ERROR;
  registerMemCleanup(data);
  coin__Transaction* ptransaction = (coin__Transaction*)data;
  int i;
  int result = SKY_OK;
  for( i = 0; i < n; i++){
    coin__Transaction* p = makeTransaction();
    memcpy(ptransaction, p, sizeof(coin__Transaction));
    ptransaction++;
  }
  transactions->data = data;
  transactions->len = n;
  transactions->cap = n;
  return result;
}

coin__Transaction* copyTransaction(coin__Transaction* pt1){
  Transaction__Handle handle = 0;
  coin__Transaction* ptransaction = NULL;
  int result = 0;
  result = SKY_coin_Transaction_Copy(pt1, &handle);
  cr_assert(result == SKY_OK);
  registerHandleClose(handle);
  result = SKY_coin_Get_Transaction_Object( handle, &ptransaction );
  cr_assert(result == SKY_OK, "SKY_coin_Get_Transaction_Object failed");
  return ptransaction;
}

void makeRandHash(cipher__SHA256* phash){
  GoSlice slice;
  memset(&slice, 0, sizeof(GoSlice));

  int result = SKY_cipher_RandByte( 128, (coin__UxArray*)&slice );
  cr_assert(result == SKY_OK, "SKY_cipher_RandByte failed");
  registerMemCleanup( slice.data );
  result = SKY_cipher_SumSHA256( slice, phash );
  cr_assert(result == SKY_OK, "SKY_cipher_SumSHA256 failed");
}

int makeUxArray(coin__UxArray* parray, int n){
  parray->data = malloc( sizeof(coin__UxOut) * n );
  if(!parray->data)
    return SKY_ERROR;
  registerMemCleanup( parray->data );
  parray->cap = parray->len = n;
  coin__UxOut* p = (coin__UxOut*)parray->data;
  int result = SKY_OK;
  for(int i = 0; i < n; i++){
    result = makeUxOut(p);
    if( result != SKY_OK )
      break;
    p++;
  }
  return result;
}
