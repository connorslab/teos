//! Fields needed to establish whether historical block data is available.
//!
//! XBT Knots does not expose Bitcoin Core's legacy `difficulty` field here.
//! Require the pruning fields explicitly rather than deserializing unrelated
//! fields through bitcoincore-rpc's Bitcoin-specific response type.
use serde::Deserialize;

#[derive(Deserialize)]
pub struct BackendInfo {
    pruned: bool,
    pruneheight: Option<u64>,
}

impl BackendInfo {
    pub fn prune_height(self) -> Result<Option<u64>, &'static str> {
        match (self.pruned, self.pruneheight) {
            (true, Some(height)) => Ok(Some(height)),
            (true, None) => Err("pruned backend did not report pruneheight"),
            (false, None) => Ok(None),
            (false, Some(_)) => Err("unpruned backend unexpectedly reported pruneheight"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn accepts_xbt_response_without_legacy_difficulty() {
        let info: BackendInfo = serde_json::from_str(r#"{"chain":"main","pruned":false,"blocks":978800}"#).unwrap();
        assert_eq!(info.prune_height(), Ok(None));
    }
    #[test]
    fn requires_consistent_pruning_information() {
        assert!(serde_json::from_str::<BackendInfo>(r#"{"blocks":978800}"#).is_err());
        let missing: BackendInfo = serde_json::from_str(r#"{"pruned":true}"#).unwrap();
        assert!(missing.prune_height().is_err());
        let present: BackendInfo = serde_json::from_str(r#"{"pruned":true,"pruneheight":978000}"#).unwrap();
        assert_eq!(present.prune_height(), Ok(Some(978000)));
        let inconsistent: BackendInfo = serde_json::from_str(r#"{"pruned":false,"pruneheight":978000}"#).unwrap();
        assert!(inconsistent.prune_height().is_err());
    }
}
