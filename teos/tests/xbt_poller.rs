use bitcoin::{consensus, BlockHash, Network, Work};
use lightning_block_sync::{AsyncBlockSourceResult, BlockData, BlockHeaderData, BlockSource, BlockSourceError};
use lightning_block_sync::poll::{ChainPoller, Poll, Validate};

struct FixtureSource(BlockHeaderData);
impl BlockSource for FixtureSource {
    fn get_header<'a>(&'a self, hash: &'a BlockHash, _: Option<u32>) -> AsyncBlockSourceResult<'a, BlockHeaderData> {
        Box::pin(async move {
            if self.0.header.block_hash() != *hash { return Err(BlockSourceError::persistent("wrong fixture")); }
            Ok(self.0)
        })
    }
    fn get_block<'a>(&'a self, _: &'a BlockHash) -> AsyncBlockSourceResult<'a, BlockData> {
        Box::pin(async { Err(BlockSourceError::persistent("header fixture only")) })
    }
    fn get_best_block(&self) -> AsyncBlockSourceResult<'_, (BlockHash, Option<u32>)> {
        Box::pin(async { Ok((self.0.header.block_hash(), Some(self.0.height))) })
    }
}

#[tokio::test]
async fn validates_xbt_activation_retarget_and_chainwork_links() {
    let rows: serde_json::Value = serde_json::from_str(include_str!("data/xbt-headers.json")).unwrap();
    let decode = |row: &serde_json::Value| BlockHeaderData {
        header: consensus::deserialize(&hex::decode(row["raw"].as_str().unwrap()).unwrap()).unwrap(),
        height: row["height"].as_u64().unwrap() as u32,
        chainwork: Work::from_be_bytes(hex::decode(row["verbose"]["chainwork"].as_str().unwrap()).unwrap().try_into().unwrap()),
    };
    let mut checked = 0;
    for pair in rows.as_array().unwrap().windows(2) {
        let previous = decode(&pair[0]);
        let next = decode(&pair[1]);
        if next.height != previous.height + 1 { continue; }
        let source = FixtureSource(previous);
        let poller = ChainPoller::new(&source, Network::Bitcoin);
        let valid = next.validate(next.header.block_hash()).unwrap();
        assert_eq!(poller.look_up_previous_header(&valid).await.unwrap().header, previous.header);
        let mut wrong_work = next;
        wrong_work.chainwork = previous.chainwork;
        let invalid = wrong_work.validate(next.header.block_hash()).unwrap();
        assert!(poller.look_up_previous_header(&invalid).await.is_err());
        let mut wrong_height = next;
        wrong_height.height += 1;
        let invalid = wrong_height.validate(next.header.block_hash()).unwrap();
        assert!(poller.look_up_previous_header(&invalid).await.is_err());
        checked += 1;
    }
    assert_eq!(checked, 3, "must cover activation, retarget boundary, and ordinary XBT blocks");
}
