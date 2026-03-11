# OpenAPI Command Plan

| Tag | Operation ID | Endpoint | Suggested CLI Command | Request Body |
| --- | --- | --- | --- | --- |
| `Admin - Driver` | `adminGetDriverInfo` | `GET /api/admin/driver/info` | `openlist-cli call adminGetDriverInfo` | no |
| `Admin - Driver` | `adminListDriverInfo` | `GET /api/admin/driver/list` | `openlist-cli call adminListDriverInfo` | no |
| `Admin - Driver` | `adminListDriverNames` | `GET /api/admin/driver/names` | `openlist-cli call adminListDriverNames` | no |
| `Admin - Index` | `adminBuildIndex` | `POST /api/admin/index/build` | `openlist-cli call adminBuildIndex` | yes |
| `Admin - Index` | `adminClearIndex` | `POST /api/admin/index/clear` | `openlist-cli call adminClearIndex` | no |
| `Admin - Index` | `adminGetIndexProgress` | `GET /api/admin/index/progress` | `openlist-cli call adminGetIndexProgress` | no |
| `Admin - Index` | `adminStopIndex` | `POST /api/admin/index/stop` | `openlist-cli call adminStopIndex` | no |
| `Admin - Index` | `adminUpdateIndex` | `POST /api/admin/index/update` | `openlist-cli call adminUpdateIndex` | yes |
| `Admin - Meta` | `adminCreateMeta` | `POST /api/admin/meta/create` | `openlist-cli call adminCreateMeta` | yes |
| `Admin - Meta` | `adminDeleteMeta` | `POST /api/admin/meta/delete` | `openlist-cli call adminDeleteMeta` | no |
| `Admin - Meta` | `adminGetMeta` | `GET /api/admin/meta/get` | `openlist-cli call adminGetMeta` | no |
| `Admin - Meta` | `adminListMetas` | `GET /api/admin/meta/list` | `openlist-cli call adminListMetas` | no |
| `Admin - Meta` | `adminUpdateMeta` | `POST /api/admin/meta/update` | `openlist-cli call adminUpdateMeta` | yes |
| `Admin - Offline Download` | `adminSetAria2` | `POST /api/admin/setting/set_aria2` | `openlist-cli call adminSetAria2` | yes |
| `Admin - Offline Download` | `adminSetQbittorrent` | `POST /api/admin/setting/set_qbit` | `openlist-cli call adminSetQbittorrent` | yes |
| `Admin - Offline Download` | `adminSetTransmission` | `POST /api/admin/setting/set_transmission` | `openlist-cli call adminSetTransmission` | yes |
| `Admin - Scan` | `adminGetScanProgress` | `GET /api/admin/scan/progress` | `openlist-cli call adminGetScanProgress` | no |
| `Admin - Scan` | `adminStartScan` | `POST /api/admin/scan/start` | `openlist-cli call adminStartScan` | no |
| `Admin - Scan` | `adminStopScan` | `POST /api/admin/scan/stop` | `openlist-cli call adminStopScan` | no |
| `Admin - Setting` | `adminDefaultSettings` | `POST /api/admin/setting/default` | `openlist-cli call adminDefaultSettings` | no |
| `Admin - Setting` | `adminDeleteSetting` | `POST /api/admin/setting/delete` | `openlist-cli call adminDeleteSetting` | no |
| `Admin - Setting` | `adminGetSetting` | `GET /api/admin/setting/get` | `openlist-cli call adminGetSetting` | no |
| `Admin - Setting` | `adminListSettings` | `GET /api/admin/setting/list` | `openlist-cli call adminListSettings` | no |
| `Admin - Setting` | `adminResetToken` | `POST /api/admin/setting/reset_token` | `openlist-cli call adminResetToken` | no |
| `Admin - Setting` | `adminSaveSettings` | `POST /api/admin/setting/save` | `openlist-cli call adminSaveSettings` | yes |
| `Admin - Storage` | `adminCreateStorage` | `POST /api/admin/storage/create` | `openlist-cli call adminCreateStorage` | yes |
| `Admin - Storage` | `adminDeleteStorage` | `POST /api/admin/storage/delete` | `openlist-cli call adminDeleteStorage` | no |
| `Admin - Storage` | `adminDisableStorage` | `POST /api/admin/storage/disable` | `openlist-cli call adminDisableStorage` | no |
| `Admin - Storage` | `adminEnableStorage` | `POST /api/admin/storage/enable` | `openlist-cli call adminEnableStorage` | no |
| `Admin - Storage` | `adminGetStorage` | `GET /api/admin/storage/get` | `openlist-cli call adminGetStorage` | no |
| `Admin - Storage` | `adminListStorages` | `GET /api/admin/storage/list` | `openlist-cli call adminListStorages` | no |
| `Admin - Storage` | `adminLoadAllStorages` | `POST /api/admin/storage/load_all` | `openlist-cli call adminLoadAllStorages` | no |
| `Admin - Storage` | `adminUpdateStorage` | `POST /api/admin/storage/update` | `openlist-cli call adminUpdateStorage` | yes |
| `Admin - User` | `adminCancel2FA` | `POST /api/admin/user/cancel_2fa` | `openlist-cli call adminCancel2FA` | no |
| `Admin - User` | `adminCreateUser` | `POST /api/admin/user/create` | `openlist-cli call adminCreateUser` | yes |
| `Admin - User` | `adminDelUserCache` | `POST /api/admin/user/del_cache` | `openlist-cli call adminDelUserCache` | no |
| `Admin - User` | `adminDeleteUser` | `POST /api/admin/user/delete` | `openlist-cli call adminDeleteUser` | no |
| `Admin - User` | `adminGetUser` | `GET /api/admin/user/get` | `openlist-cli call adminGetUser` | no |
| `Admin - User` | `adminListUsers` | `GET /api/admin/user/list` | `openlist-cli call adminListUsers` | no |
| `Admin - User` | `adminUpdateUser` | `POST /api/admin/user/update` | `openlist-cli call adminUpdateUser` | yes |
| `Archive` | `fsArchiveDecompress` | `POST /api/fs/archive/decompress` | `openlist-cli call fsArchiveDecompress` | yes |
| `Archive` | `fsArchiveList` | `POST /api/fs/archive/list` | `openlist-cli call fsArchiveList` | yes |
| `Archive` | `fsArchiveMeta` | `POST /api/fs/archive/meta` | `openlist-cli call fsArchiveMeta` | yes |
| `Auth` | `currentUser` | `GET /api/me` | `openlist-cli call currentUser` | no |
| `Auth` | `generate2FA` | `POST /api/auth/2fa/generate` | `openlist-cli call generate2FA` | no |
| `Auth` | `login` | `POST /api/auth/login` | `openlist-cli call login` | yes |
| `Auth` | `loginHash` | `POST /api/auth/login/hash` | `openlist-cli call loginHash` | yes |
| `Auth` | `loginLdap` | `POST /api/auth/login/ldap` | `openlist-cli call loginLdap` | yes |
| `Auth` | `logout` | `GET /api/auth/logout` | `openlist-cli call logout` | no |
| `Auth` | `updateCurrentUser` | `POST /api/me/update` | `openlist-cli call updateCurrentUser` | yes |
| `Auth` | `verify2FA` | `POST /api/auth/2fa/verify` | `openlist-cli call verify2FA` | yes |
| `FileSystem` | `addOfflineDownload` | `POST /api/fs/add_offline_download` | `openlist-cli call addOfflineDownload` | yes |
| `FileSystem` | `fsBatchRename` | `POST /api/fs/batch_rename` | `openlist-cli call fsBatchRename` | yes |
| `FileSystem` | `fsCopy` | `POST /api/fs/copy` | `openlist-cli call fsCopy` | yes |
| `FileSystem` | `fsDirs` | `POST /api/fs/dirs` | `openlist-cli call fsDirs` | yes |
| `FileSystem` | `fsFormUpload` | `PUT /api/fs/form` | `openlist-cli call fsFormUpload` | yes |
| `FileSystem` | `fsGet` | `POST /api/fs/get` | `openlist-cli call fsGet` | yes |
| `FileSystem` | `fsGetDirectUploadInfo` | `POST /api/fs/get_direct_upload_info` | `openlist-cli call fsGetDirectUploadInfo` | yes |
| `FileSystem` | `fsLink` | `POST /api/fs/link` | `openlist-cli call fsLink` | yes |
| `FileSystem` | `fsList` | `POST /api/fs/list` | `openlist-cli call fsList` | yes |
| `FileSystem` | `fsMkdir` | `POST /api/fs/mkdir` | `openlist-cli call fsMkdir` | yes |
| `FileSystem` | `fsMove` | `POST /api/fs/move` | `openlist-cli call fsMove` | yes |
| `FileSystem` | `fsOther` | `POST /api/fs/other` | `openlist-cli call fsOther` | yes |
| `FileSystem` | `fsRecursiveMove` | `POST /api/fs/recursive_move` | `openlist-cli call fsRecursiveMove` | yes |
| `FileSystem` | `fsRegexRename` | `POST /api/fs/regex_rename` | `openlist-cli call fsRegexRename` | yes |
| `FileSystem` | `fsRemove` | `POST /api/fs/remove` | `openlist-cli call fsRemove` | yes |
| `FileSystem` | `fsRemoveEmptyDirectory` | `POST /api/fs/remove_empty_directory` | `openlist-cli call fsRemoveEmptyDirectory` | yes |
| `FileSystem` | `fsRename` | `POST /api/fs/rename` | `openlist-cli call fsRename` | yes |
| `FileSystem` | `fsSearch` | `POST /api/fs/search` | `openlist-cli call fsSearch` | yes |
| `FileSystem` | `fsStreamUpload` | `PUT /api/fs/put` | `openlist-cli call fsStreamUpload` | yes |
| `Public` | `archiveExtensions` | `GET /api/public/archive_extensions` | `openlist-cli call archiveExtensions` | no |
| `Public` | `offlineDownloadTools` | `GET /api/public/offline_download_tools` | `openlist-cli call offlineDownloadTools` | no |
| `Public` | `ping` | `GET /ping` | `openlist-cli call ping` | no |
| `Public` | `publicSettings` | `GET /api/public/settings` | `openlist-cli call publicSettings` | no |
| `SSHKey` | `addMySSHKey` | `POST /api/me/sshkey/add` | `openlist-cli call addMySSHKey` | yes |
| `SSHKey` | `deleteMySSHKey` | `POST /api/me/sshkey/delete` | `openlist-cli call deleteMySSHKey` | no |
| `SSHKey` | `listMySSHKeys` | `GET /api/me/sshkey/list` | `openlist-cli call listMySSHKeys` | no |
| `Sharing` | `createSharing` | `POST /api/share/create` | `openlist-cli call createSharing` | yes |
| `Sharing` | `deleteSharing` | `POST /api/share/delete` | `openlist-cli call deleteSharing` | no |
| `Sharing` | `disableSharing` | `POST /api/share/disable` | `openlist-cli call disableSharing` | no |
| `Sharing` | `enableSharing` | `POST /api/share/enable` | `openlist-cli call enableSharing` | no |
| `Sharing` | `getSharing` | `GET /api/share/get` | `openlist-cli call getSharing` | no |
| `Sharing` | `listSharings` | `POST /api/share/list` | `openlist-cli call listSharings` | yes |
| `Sharing` | `updateSharing` | `POST /api/share/update` | `openlist-cli call updateSharing` | yes |
