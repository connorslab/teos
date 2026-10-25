use bitcoin::{block::Header, consensus, hashes::Hash, BlockHash, Work};
use std::str::FromStr;

#[test]
fn full_xbt_block_matches_header_and_merkle_commitments() {
    let raw = hex::decode(include_str!("data/xbt-block-978800.hex").trim()).unwrap();
    let block: bitcoin::Block = consensus::deserialize(&raw).unwrap();
    assert_eq!(block.block_hash().to_string(), "000000000000000078251b55f99670d8de8141ca4c49819e1f3e1981cac20873");
    assert_eq!(consensus::serialize(&block), raw);
    assert!(block.check_merkle_root());
    assert!(block.check_witness_commitment());
    let mut altered = block;
    altered.txdata.pop();
    assert!(!altered.check_merkle_root());
}

#[test]
fn knots_headers_roundtrip_hash_and_proof_of_work() {
    let vectors: serde_json::Value = serde_json::from_str(include_str!("data/xbt-headers.json")).unwrap();
    for row in vectors.as_array().unwrap() {
        let raw = hex::decode(row["raw"].as_str().unwrap()).unwrap();
        let header: Header = consensus::deserialize(&raw).unwrap();
        let expected = BlockHash::from_str(row["hash"].as_str().unwrap()).unwrap();
        assert_eq!(consensus::serialize(&header), raw);
        assert_eq!(header.block_hash(), expected);
        assert_eq!(header.validate_pow(header.target()).unwrap(), expected);
        assert_eq!(header.blake2b.is_some(), row["height"].as_u64().unwrap() >= 961640);
        if let Some(extra) = header.blake2b {
            assert_eq!(extra.height as u64, row["height"].as_u64().unwrap());
        }
        let mut altered = header;
        altered.nonce ^= 1;
        assert_ne!(altered.block_hash(), expected);
        let mut trailing = raw.clone();
        trailing.push(0);
        assert!(consensus::deserialize::<Header>(&trailing).is_err());
    }
    for pair in vectors.as_array().unwrap().windows(2) {
        if pair[1]["height"].as_u64().unwrap() != pair[0]["height"].as_u64().unwrap() + 1 { continue; }
        let next: Header = consensus::deserialize(&hex::decode(pair[1]["raw"].as_str().unwrap()).unwrap()).unwrap();
        assert_eq!(next.prev_blockhash, BlockHash::from_str(pair[0]["hash"].as_str().unwrap()).unwrap());
        let work = |r: &serde_json::Value| Work::from_be_bytes(hex::decode(r["verbose"]["chainwork"].as_str().unwrap()).unwrap().try_into().unwrap());
        assert_eq!(work(&pair[1]), work(&pair[0]) + next.work());
    }
    assert_ne!(BlockHash::all_zeros(), BlockHash::from_str(vectors[0]["hash"].as_str().unwrap()).unwrap());
}
