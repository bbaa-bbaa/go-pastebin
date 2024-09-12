(function () {
  const $ = mdui.$;
  function isDesktop() {
    return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) === false;
  }
  $(function () {
    (function useradd() {
      let useradd_form = $("#useradd");
      useradd_form.on("submit", function () {
        const email = $("#admin-email-text")[0].value;
        const username = $("#admin-username-text")[0].value;
        const password = $("#admin-password-text")[0].value;
        const group = $("#admin-group-text")[0].value;

        if (!email || !username || !password || !group) {
          mdui.alert("请填写所有字段");
          return;
        }

        const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
        if (!emailRegex.test(email)) {
          mdui.alert("请输入有效的电子邮件地址");
          return;
        }

        $.ajax({
          method: "POST",
          url: "/api/user/add",
          data: JSON.stringify({
            email: email,
            username: username,
            password: password,
            group: group
          }),
          contentType: "application/json"
        }).then(res => {
          let response = JSON.parse(res);
          if (response.code != 0) {
            mdui.alert(response.error);
          } else {
            mdui.alert("添加成功");
          }
        });
      })
    })();

    (function paste_manage() {
      const paste_viewer_back_to_query = $(".paste-viewer-back-to-query");

      const paste_manage_pastes = $("#paste-manage-pastes");
      const paste_manage_null = $("#paste-manage-null");

      const paste_manage_progress = $(".paste-manage-progress");
      const paste_manage_prev = $("#paste-manage-prev");
      const paste_manage_next = $("#paste-manage-next");
      const paste_manage_pager_hint = $("#paste-manage-pager-hint");
      const paste_manage_pager = $(".paste-manage-pager");

      const paste_manage_mdui_panel = new mdui.Panel("#paste-manage-pastes .mdui-panel");
      const paste_manage_panel = $("#paste-manage-pastes .mdui-panel");

      let page = 1;
      let max_page = 1;
      let paste_total = 0;
      const page_size = 10;

      let paste_manager_map = new Map();
      let paste_detail_opened = new Set(); // uuid
      const paste_detail_opened_limit = 20 * page_size;

      function pager_check() {
        if (page == 1) {
          paste_manage_prev.attr("disabled", "disabled");
        } else {
          paste_manage_prev.removeAttr("disabled");
        }
        if (page >= max_page) {
          paste_manage_next.attr("disabled", "disabled");
        } else {
          paste_manage_next.removeAttr("disabled");
        }
        paste_manage_pager_hint.text(`第 ${page} 页 / 共 ${max_page} 页`);
      }

      paste_viewer_back_to_query.on("click", function () {
        if (paste_viewer_back_to_manage) {
          setTimeout(() => {
            paste_app_tab.show(2);
            paste_viewer_back_to_query.trigger("pastebin.viewer.clean");
          }, 600);
        }
      });

      function register_action_button(panel, paste_uuid, paste_hash) {
        const paste_manage_delete_btn = panel.find(".paste-manage-delete-btn");
        const paste_manage_view_btn = panel.find(".paste-manage-view-btn");
        const paste_manage_edit_btn = panel.find(".paste-manage-edit-btn");
        const paste_manage_copy_url_btn = panel.find(".paste-manage-copy-url-btn");
        let uuid = paste_uuid;
        let hash = paste_hash;

        panel.on("open.mdui.panel", function () {
          paste_detail_opened.add(uuid);
          if (paste_detail_opened.size > paste_detail_opened_limit) {
            paste_detail_opened.delete(paste_detail_opened.values().next().value);
          }
        });

        panel.on("close.mdui.panel", function () {
          paste_detail_opened.delete(uuid);
        });


        paste_manage_view_btn.on("click", function (e) {
          paste_manage_progress.show();
          paste_manage_view_btn.attr("disabled", "disabled");
          $.ajax({
            method: "GET",
            url: "../api/paste/" + uuid,
            headers: {
              Accept: "application/json"
            },
            complete: function (xhr) {
              let response = JSON.parse(xhr.responseText);
              if (xhr.status == 200 && response.code === 0) {
                location.href="../#"+hash;
              } else {
                mdui.snackbar("加载失败: " + response.error);
              }
              paste_manage_view_btn.removeAttr("disabled");
              paste_manage_progress.hide();
            }
          });
        });

        paste_manage_copy_url_btn.on("click", function () {
          let element = panel.find(".paste-link > a");
          let url = element.attr("href");
          function selectAndHint() {
            let selection = window.getSelection();
            let range = document.createRange();
            range.selectNodeContents(element.get(0));
            selection.removeAllRanges();
            selection.addRange(range);
            mdui.snackbar("请按 Ctrl+C 复制");
          }
          if (navigator.clipboard) {
            navigator.clipboard
              .writeText(url)
              .then(() => {
                mdui.snackbar("已复制到剪贴板");
              })
              .catch(err => {
                selectAndHint();
              });
          } else {
            selectAndHint();
          }
        });

        paste_manage_delete_btn.on("click", function (e) {
          paste_manage_progress.show();
          paste_manage_delete_btn.attr("disabled", "disabled");
          $.ajax({
            method: "DELETE",
            url: "../" + uuid+"?force=true",
            headers: {
              Accept: "application/json"
            },
            complete: function (xhr) {
              let response = JSON.parse(xhr.responseText);
              if (xhr.status == 200 && response.code === 0) {
                mdui.snackbar("删除成功");
              } else {
                mdui.snackbar("删除失败: " + response.error);
              }
              paste_manage_delete_btn.removeAttr("disabled");
              paste_manage_progress.hide();
            }
          });
        });

        paste_manage_edit_btn.on("click", function (e) {
          location.href="../?uuid="+uuid;
        });
      }

      function generate_paste_panel_html(paste) {
        let now = new Date().getTime();
        let expired = new Date(paste.expire_after).getTime();
        let pastes_panel = `
          <div class="mdui-panel-item${paste_detail_opened.has(paste.uuid) ? " mdui-panel-item-open" : ""}">
            <div class="mdui-panel-item-header">
        `
        pastes_panel += `<div class="mdui-m-r-1">`;
        let file_type = paste.mime_type.split("/")[0] || "application";
        if (file_type == "image") {
          pastes_panel += `<i class="mdui-icon material-icons">image</i>`;
        } else if (file_type == "audio") {
          pastes_panel += `<i class="mdui-icon material-icons">audiotrack</i>`;
        } else if (file_type == "video") {
          pastes_panel += `<i class="mdui-icon material-icons">videocam</i>`;
        } else if (file_type == "text") {
          pastes_panel += `<i class="mdui-icon material-icons">menu</i>`;
        } else {
          pastes_panel += `<i class="mdui-icon material-icons">insert_drive_file</i>`;
        }
        pastes_panel += `</div>`;
        if (paste.filename != "" && paste.filename != "-") {
          pastes_panel += `<div class="mdui-panel-item-title" style="overflow: visible;">${paste.filename}</div>`;
        } else {
          pastes_panel += `<div class="mdui-panel-item-title" style="overflow: visible;">${paste.short_url || paste.hash}</div>`;
        }
        if (paste.user) {
          pastes_panel += `
          <div class="mdui-panel-item-summary mdui-invisible-xs-down">User: ${paste.user.username}</div>
        `;
        }
        pastes_panel += `
          <div class="mdui-panel-item-summary mdui-invisible-xs-down">Time: ${paste.created_at.substring(0, Math.min(23, paste.created_at.length))}</div>
        `;
        pastes_panel += `
              <i class="mdui-panel-item-arrow mdui-icon material-icons">keyboard_arrow_down</i>
            </div>
            <div class="mdui-panel-item-body">
              <div class="raw-result">
                <button class="mdui-btn mdui-btn-icon mdui-ripple paste-manage-copy-url-btn mdui-float-right">
                  <i class="mdui-icon material-icons">content_copy</i>
                </button>
        `
        if (paste.user) {
          pastes_panel += `
                <p><strong>user:</strong> ${paste.user.username} [uid: ${paste.uid}](email: ${paste.user.email})</p>
          `;
        } else {
          pastes_panel += `
                <p><strong>uid:</strong> ${paste.uid}</p>
          `;
        }
        pastes_panel += `
                <p><strong>date:</strong> ${paste.created_at}</p>
        `;
        if (paste.expire_after != "0001-01-01T00:00:00Z") {
          if (!paste.delete_if_not_available) {
            if (expired > now) {
              pastes_panel += ` <p><strong>expire:</strong> ${paste.expire_after}</p>`;
            } else {
              pastes_panel += ` <p><strong>expire:</strong> ${paste.expire_after} <span class="mdui-text-color-red">(expired)</span></p>`;
            }
          } else {
            if (expired > now) {
              pastes_panel += ` <p><strong>expire:</strong> ${paste.expire_after} (auto delete)</p>`;
            } else {
              pastes_panel += ` <p><strong>expire:</strong> ${paste.expire_after} <span class="mdui-text-color-red">(expired, delete flagged)</span></p>`;
            }
          }
        } else {
          pastes_panel += ` <p><strong>expire:</strong> never</p>`;
        }
        pastes_panel += `
                <p><strong>digest:</strong> ${paste.digest}</p>
                <p><strong>long:</strong> ${paste.hash}</p>
                <p><strong>short:</strong> ${paste.short_url}</p>
                <p><strong>filename:</strong> ${paste.filename}</p>
                <p><strong>mime:</strong> ${paste.mime_type}</p>
                <p><strong>size:</strong> ${paste.size}</p>
        `;
        if (paste.max_access_count == 0) {
          pastes_panel += `<p><strong>access_count:</strong> ${paste.access_count} (max: nolimit)</p>`;
        } else {
          if (!paste.delete_if_not_available) {
            if (paste.access_count < paste.max_access_count) {
              pastes_panel += `<p><strong>access_count:</strong> ${paste.access_count} (max: ${paste.max_access_count})</p>`;
            } else {
              pastes_panel += `<p><strong>access_count:</strong> ${paste.access_count} (max: ${paste.max_access_count}) <span class="mdui-text-color-red">(limit reached)</span></p>`;
            }
          } else {
            if (paste.access_count < paste.max_access_count) {
              pastes_panel += `<p><strong>access_count:</strong> ${paste.access_count} (max: ${paste.max_access_count}) (auto delete)</p>`;
            } else {
              pastes_panel += `<p><strong>access_count:</strong> ${paste.access_count} (max: ${paste.max_access_count}) <span class="mdui-text-color-red">(limit reached, delete flagged)</span></p>`;
            }
          }
        }
        pastes_panel += `
                <p><strong>password:</strong> ${paste.has_password ? "yes" : "no"}</p>
                <p><strong>uuid:</strong> ${paste.uuid}</p>
        `;
        if (paste.delete_if_not_available) {
          if ((paste.expire_after != "0001-01-01T00:00:00Z" && now >= expired) || (paste.max_access_count != 0 && paste.access_count >= paste.max_access_count)) {
            pastes_panel += ` <p class="mdui-text-color-red"><strong>hold_before:</strong> ${paste.hold_before} (count: ${paste.hold_count})</p>`;
          }
        }
        pastes_panel += `
              </div>
              <div>
                <p class="paste-link">url: <a href="${paste.url}" target="${isDesktop() ? "_blank" : "_self"}">${paste.url}</a></p>
              </div>
              <div class="mdui-panel-item-actions">
                <div class="mdui-container-fluid">
                  <div class="mdui-row">
                    <div class="mdui-col-sm-3 mdui-col-md-6 mdui-col-lg-9 mdui-hidden-xs"></div>
                    <div class="mdui-col-lg-1 mdui-col-md-2 mdui-col-sm-3 mdui-col-xs-4">
                      <button class="mdui-btn mdui-btn-block mdui-ripple mdui-color-red paste-manage-delete-btn" style="min-width: 0;">删除</button>
                    </div>
                    <div class="mdui-col-lg-1 mdui-col-md-2 mdui-col-sm-3 mdui-col-xs-4">
                      <button class="mdui-btn mdui-btn-block mdui-ripple mdui-color-blue-accent paste-manage-view-btn" style="min-width: 0;">查看</button>
                    </div>
                    <div class="mdui-col-lg-1 mdui-col-md-2 mdui-col-sm-3 mdui-col-xs-4">
                      <button class="mdui-btn mdui-btn-block mdui-ripple mdui-color-theme-accent paste-manage-edit-btn" style="min-width: 0;">编辑</button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        `;
        return pastes_panel;
      }

      function list_paste(scrollOffset) {
        paste_manage_progress.show();
        paste_manage_pager.attr("disabled", "disabled");
        $.ajax({
          method: "GET",
          url: "../api/paste/pastes",
          headers: {
            Accept: "application/json"
          },
          data: {
            page: page,
            page_size: page_size
          },
          complete: function (xhr) {
            let response = JSON.parse(xhr.responseText);
            if (xhr.status == 200 && response.code === 0) {
              paste_total = response.total;
              max_page = Math.ceil(paste_total / page_size);
              if (response.pastes.length != 0) {
                let paste_map = new Map();
                for (let paste of response.pastes) {
                  if (response.users[paste.uid]) {
                    paste.user = response.users[paste.uid];
                  }
                  paste_map.set(paste.uuid, paste);
                }
                for (let [uuid, paste] of paste_manager_map) {
                  if (!paste_map[uuid]) {
                    paste.panel.remove();
                    paste_manager_map.delete(uuid);
                  } else if (!_.isEqual(paste.paste, paste_map[uuid])) {
                    let panel = $(generate_paste_panel_html(paste_map[uuid]));
                    paste.panel.replaceWith(panel);
                    paste_manager_map.set(uuid, { paste: paste_map[uuid], panel });
                    register_action_button(panel, paste.uuid, paste.hash);
                  }
                }
                // add new pastes
                let start_node = null;
                for (let [index, paste] of response.pastes.entries()) {
                  if (!paste_manager_map.has(paste.uuid)) {
                    let panel = $(generate_paste_panel_html(paste));
                    paste_manager_map.set(paste.uuid, { paste, panel });
                    if (start_node != null) {
                      start_node.after(panel);
                      start_node = panel;
                    } else if (index == 0) {
                      paste_manage_panel.prepend(panel);
                      start_node = panel
                    } else {
                      paste_manage_panel.append(panel);
                    }
                    register_action_button(panel, paste.uuid, paste.hash);

                  }
                }
                mdui.mutation(); // re-render

                paste_manage_null.hide();
                paste_manage_pastes.show();
              } else {
                paste_manage_pastes.hide();
                paste_manage_null.show();
              }
            }
            paste_manage_progress.hide();
            pager_check();
            if (scrollOffset) {
              window.scrollTo(0, document.documentElement.scrollHeight - scrollOffset);
            }
          }
        });
      }

      paste_manage_prev.on("click", function () {
        if (page > 1) {
          page--;
          list_paste(document.documentElement.scrollHeight - document.documentElement.scrollTop);
        }
      });

      paste_manage_next.on("click", function () {
        if (page < max_page) {
          page++;
          list_paste(document.documentElement.scrollHeight - document.documentElement.scrollTop);
        }
      });

      paste_manage_pastes.hide();
      paste_manage_null.show();
      list_paste();
    })();
  });

})();