(function () {
  let query_hash = window.location.hash.replace(/^#/, "");
  let $ = mdui.$;
  function easeInSine(t) {
    return 1 - Math.cos((t * Math.PI) / 2);
  }

  function isDesktop() {
    return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) === false;
  }

  let user_info = null;
  let paste_force_delete = false;

  function formatSize(bytes) {
    const units = ["Bytes", "KiB", "MiB", "GiB", "TiB"];
    let unitIndex = 0;
    while (bytes >= 1024 && unitIndex < units.length - 1) {
      bytes /= 1024;
      unitIndex++;
    }
    return `${bytes.toFixed(2)} ${units[unitIndex]}`;
  }

  function updateUserPasteSize() {
    $.ajax({
      method: "GET",
      url: "api/user/size",
      contentType: "application/json",
      success: function (response) {
        response = JSON.parse(response || "{}");
        if (response.code === 0) {
          const totalSize = formatSize(response.total_size);
          $("#total-paste-size").text(`Paste总大小: ${totalSize}`);
        } else {
          console.error("Error fetching total paste size:", response.error);
        }
      },
      error: function (xhr, status, error) {
        console.error("Error fetching total paste size:", error);
      }
    });
  }

  $(function () {
    document.body.addEventListener("drop", function (e) {
      e.preventDefault();
      e.stopPropagation();
    });
    document.body.addEventListener("dragover", function (e) {
      e.preventDefault();
    });

    const container = $("body > div.mdui-container").get(0);
    let config = {
      allow_anonymous: document.querySelector("meta[name='x-allow-anonymous']").content === "true"
    };
    const paste_app_tab = new mdui.Tab("#paste-mdui-tab");
    const paste_app_tab_element = $("#paste-mdui-tab");
    paste_app_tab_element.addClass("paste-tab-loaded");
    const new_paste_tab = $("#new-paste-tab");
    const paste_viewer_tab = $("#paste-viewer-tab");
    const paste_manage_tab = $("#paste-manage-tab");
    let user_is_login;
    function update_user_info(preloaded_user_info) {
      let login_request;
      if (preloaded_user_info) {
        user_info = preloaded_user_info;
        login_request = Promise.resolve(true);
      } else {
        login_request = $.ajax({
          method: "GET",
          url: "api/user",
          contentType: "application/json"
        })
          .then(res => {
            let response = JSON.parse(res);
            if (response.code != 0) {
              return false;
            }
            user_info = response.info;
            updateUserPasteSize();
            return true;
          })
          .catch(() => false);
      }
      user_is_login = login_request.then(is_login => {
        if (is_login) {
          new_paste_tab.removeAttr("disabled");
          paste_manage_tab.show();
        } else {
          user_info = null;
          paste_manage_tab.hide();
          if (!config.allow_anonymous) {
            new_paste_tab.attr("disabled", "disabled");
            paste_viewer_tab.show();
            if (paste_app_tab.activeIndex == 0) {
              paste_app_tab.activeIndex = 1;
            }
          }
        }
        paste_app_tab.show(paste_app_tab.activeIndex);
        return is_login;
      });
      return user_is_login;
    }

    new_paste_tab.on("click", function (e) {
      if (new_paste_tab.attr("disabled")) {
        mdui.snackbar("此 Pastebin 仅允许登录用户创建 Paste");
      }
    });

    update_user_info();

    function Collapse(jq) {
      this.$ = jq;
      this.element = jq.get(0);
      this.expanded = this.$.hasClass("pb-collapse-open");
      this.expanded_magrin = this.element.style.margin;
      this._callback = {};
      this._pendingCallback = {};
      if (!this.expanded) {
        this.element.style.margin = "0";
        this.$.css("height", "0px");
      }
      this.$.on("transitionstart", e => {
        if (e.target != this.element || e.propertyName != "height") {
          return;
        }
        this._callback = this._pendingCallback;
      });
      this.$.on("transitioncancel", e => {
        if (e.target != this.element || e.propertyName != "height") {
          return;
        }
        console.log("trans cancel");
        if (this._callback.reject) this._callback.reject();
      });
      this.$.on("transitionend", e => {
        if (e.target != this.element || e.propertyName != "height") {
          return;
        }
        if (this._callback.resolve) this._callback.resolve();
      });
      this.$.addClass("pb-collapse-inited");
    }

    Collapse.prototype._pendingPromise = function (resolve, reject) {
      this._pendingCallback = { resolve, reject };
    };

    Collapse.prototype.close = function () {
      if (this.$.css("height") == "0px" || !this.expanded) {
        return Promise.resolve();
      }
      this.expanded = false;
      let close = new Promise((resolve, reject) => {
        this.$.css("height", this.element.scrollHeight + "px");
        requestAnimationFrame(() => {
          this._pendingPromise(resolve, reject);
          this.$.css("height", "0px");
          this.element.style.margin = "0";
        });
      });
      return close;
    };

    Collapse.prototype.open = function () {
      if (this.$.css("height") == "auto" || this.expanded) {
        return Promise.resolve();
      }
      this.expanded = true;
      let open = new Promise((resolve, reject) => {
        this._pendingPromise(resolve, reject);
        this.element.style.margin = this.expanded_magrin;
        this.$.css("height", this.element.scrollHeight + "px");
      });
      open.then(() => {
        this.$.css("height", "auto");
      });
      return open;
    };

    function CollapseGroupProxy(group, target) {
      this._group = group;
      this._target = target;
    }

    CollapseGroupProxy.prototype.open = function () {
      return this._group._open(this._target);
    };
    CollapseGroupProxy.prototype.close = function () {
      return this._group._close(this._target);
    };

    function CollapseGroup(group) {
      this._group = [];
      for (let [k, v] of Object.entries(group)) {
        let collapseManager = new Collapse(v);
        this._group.push(collapseManager);
        this[k] = new CollapseGroupProxy(this, collapseManager);
      }
    }

    CollapseGroup.prototype._open = function (target) {
      let result;
      for (let item of this._group) {
        if (item === target) {
          result = item.open();
        } else {
          item.close();
        }
      }
      return result;
    };

    CollapseGroup.prototype._close = function (target) {
      return target.close();
    };

    function testVaildURL(url) {
      return /^https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)$/im.test(url);
    }

    (function new_paste() {
      const new_paste_import_btn = $("#new-paste-import");
      const new_paste_preview_markdown_btn = $("#new-paste-preview-markdown");
      const new_paste_edit_text_btn = $("#new-paste-edit-text");
      const new_paste_markdown_preview_container = $("#new-paste-markdown-preview-container");
      const new_paste_edit_func = $("#new-paste-edit-func");

      const text_input = $("#new-paste-text-input");
      const file_input = $("#new-paste-file-input");
      const file_paste = $("#new-paste-file");
      const drop_file_overlay = $(".paste-file-drop-overlay");
      const paste_password = $("#new-paste-password");
      const paste_expire = $("#new-paste-expire");
      const paste_max_access_count = $("#new-paste-max-access-count");
      const paste_uuid = $("#new-paste-uuid");
      const paste_short_url = $("#new-paste-short-url");
      const paste_detect_mime = $("#new-paste-detect-mime");
      const paste_delete_if_not_available = $("#new-paste-delete-if-not-available");
      const paste_shorten_url = $("#new-paste-shorten-url");
      const container_detect_mime = paste_detect_mime.parent().parent("div");
      const container_paste_shorten_url = paste_shorten_url.parent().parent("div");

      const new_paste_return_container = $("#new-paste-return-container");
      const new_paste_delete_container = $("#new-paste-delete-container");
      const new_paste_update_container = $("#new-paste-update-container");
      const new_paste_submit_container = $("#new-paste-submit-container");
      const new_paste_return = $("#new-paste-return");
      const paste_delete = $("#new-paste-delete");
      const paste_update = $("#new-paste-update");
      const paste_float_update = $("#new-paste-float-update");
      const paste_submit = $("#new-paste-submit");
      const paste_load = $("#new-paste-load-from-file");

      const file_paste_icon = $("#new-paste-file-icon");
      const file_paste_preview = $("#new-paste-file-preview");
      const file_paste_filename = $("#new-paste-file-filename");
      const file_paste_progress = $("#new-paste-file-progress");
      const collapse_file_paste_progress = new Collapse(file_paste_progress);
      const file_paste_progress_bar = $("#new-paste-file-progress-bar");
      const file_paste_progress_text = $("#new-paste-file-progress-text");

      const new_paste_result = $("#new-paste-result");
      const new_paste_result_link = $("#new-paste-result-link");
      const new_paste_result_raw = $("#new-paste-result-raw");
      const new_paste_result_qr_code = $("#new-paste-result-qrcode");
      const new_paste_result_copy = $("#new-paste-result-copy");
      const new_paste_result_title = $("#new-paste-result-title");

      const short_url_error = $("#new-paste-short-url-error");

      const action_button = $(".new-paste-action-button");

      let paste_file = null; // file object

      let paste_preview_element = null;
      function paste_preview(file) {
        if (paste_preview_element) {
          let url = paste_preview_element.attr("src");
          if (url) {
            URL.revokeObjectURL(url);
          }
          paste_preview_element.remove();
        }
        if (file.type.startsWith("image/")) {
          paste_preview_element = $(`<img style="max-height: inherit; max-width:100%">`).attr("src", URL.createObjectURL(file)).appendTo(file_paste_preview);
        } else if (file.type.startsWith("audio/")) {
          paste_preview_element = $('<audio controls style="max-height: inherit; max-width:100%">').attr("src", URL.createObjectURL(file)).appendTo(file_paste_preview);
        } else if (file.type.startsWith("video/")) {
          paste_preview_element = $('<video controls style="max-height: inherit; max-width:100%">').attr("src", URL.createObjectURL(file)).appendTo(file_paste_preview);
        } else {
          paste_preview_element = null;
          file_paste_preview.hide();
          file_paste_icon.show();
          return;
        }
        file_paste_icon.hide();
        file_paste_preview.show();
      }

      let check_and_show_shorten_url = _.debounce(function () {
        if (testVaildURL(text_input.val())) {
          container_paste_shorten_url.show();
          container_detect_mime.hide();
        } else {
          container_paste_shorten_url.hide();
          container_detect_mime.show();
          paste_shorten_url.prop("checked", false);
        }
      }, 300);

      text_input.on("input", function () {
        check_and_show_shorten_url();
      });

      new_paste_preview_markdown_btn.on("click", function () {
        let text = text_input.val();
        if (text.length == 0) {
          return;
        }
        new_paste_preview_markdown_btn.hide();
        new_paste_edit_text_btn.show();
        new_paste_import_btn.hide();
        new_paste_markdown_preview_container.html(DOMPurify.sanitize(marked.parse(text)));
        new_paste_markdown_preview_container.show();
        text_input.parent().hide();
      });

      new_paste_edit_text_btn.on("click", function () {
        new_paste_edit_text_btn.hide();
        new_paste_preview_markdown_btn.show();
        let uuid = paste_uuid.val();
        if (uuid.length !== 0 && check_uuid(uuid)) {
          new_paste_import_btn.show();
        }
        text_input.parent().show();
        new_paste_markdown_preview_container.hide();
      });

      function show_file_paste_info() {
        if (!paste_file) {
          return;
        }
        file_paste_filename.text(paste_file.name + " (" + (Math.ceil((paste_file.size / 1024 / 1024) * 100) / 100).toFixed(2).toString() + " MiB)");
        paste_preview(paste_file);
        container_detect_mime.show();
        container_paste_shorten_url.hide();
        paste_shorten_url.prop("checked", false);
        paste_load.text("切换到文本模式").removeClass("mdui-color-theme-accent").addClass("mdui-color-blue-accent");
        text_input.parent().hide();
        new_paste_edit_func.hide();
        file_paste.show();
      }

      function set_paste_file(file) {
        if (!file) {
          return;
        }
        paste_file = file;
        show_file_paste_info();
      }

      function switch_to_text_paste() {
        paste_file = null;
        file_input.val("");
        paste_load.text("从文件中加载").removeClass("mdui-color-blue-accent").addClass("mdui-color-theme-accent");
        file_paste.hide();
        new_paste_edit_func.show();
        text_input.parent().show();
      }

      new_paste_import_btn.on("click", async function () {
        function getHashFromUuid(uuid) {
          return new Promise((resolve, reject) => {
            $.ajax({
              method: "GET",
              url: "api/paste/" + uuid,
              headers: {
                Accept: "application/json"
              },
              complete: function (xhr) {
                let response = JSON.parse(xhr.responseText || "{}");
                if (xhr.status == 200 && response.code === 0) {
                  resolve(response);
                } else {
                  reject(response);
                }
              }
            });
          });
        }
        function getPasteFromHash(info) {
          return new Promise((resolve, reject) => {
            $.ajax({
              method: "GET",
              url: info.hash + (info.mime_type == "application/vnd.pastebin.shorten" ? "/raw" : ""),
              complete: function (xhr) {
                if (xhr.status) {
                  resolve(xhr.responseText);
                } else {
                  reject();
                }
              }
            });
          });
        }
        let uuid = paste_uuid.val();
        if (uuid.length == 0 || !check_uuid(uuid)) {
          mdui.snackbar("无效的 UUID");
          return;
        }
        new_paste_import_btn.parent().css("pointer-events", "none");
        new_paste_import_btn.addClass("importing");
        new Promise((resolve, reject) => {
          mdui.confirm(
            "导入 Paste 会覆盖当前编辑的内容，是否继续？",
            "确认导入",
            function () {
              resolve();
            },
            function () {
              reject({ error: "用户取消" });
            },
            {
              history: false,
              modal: true
            }
          );
        })
          .then(() => getHashFromUuid(uuid))
          .then(response => {
            if (!response.info) {
              return Promise.reject();
            }
            if (
              !response.info.mime_type.startsWith("text/") &&
              !["application/javascript", "application/json", "application/xml", "application/vnd.pastebin.shorten"].includes(response.info.mime_type)
            ) {
              return Promise.reject({ error: "无法导入非支持类型的 Paste" });
            }
            if (response.info.size > 5 * 1024 * 1024) {
              return Promise.reject({ error: "无法导入大于 5 MiB 的 Paste" });
            }
            return getPasteFromHash(response.info);
          })
          .then(text => {
            text_input.val(text);
            new_paste_preview_markdown_btn.show();
            new_paste_markdown_preview_container.hide();
            text_input.get(0).dispatchEvent(new Event("input"));
            new_paste_import_btn.removeClass("importing");
            new_paste_import_btn.parent().css("pointer-events", null);
            new_paste_import_btn.addClass("mdui-color-green-600");
            setTimeout(() => {
              new_paste_import_btn.removeClass("mdui-color-green-600");
            }, 600);
          })
          .catch(response => {
            mdui.snackbar("导入时发生错误: " + (response.error || "未知错误"));
            new_paste_import_btn.removeClass("importing");
            new_paste_import_btn.parent().css("pointer-events", null);
            new_paste_import_btn.addClass("mdui-color-red-accent");
            setTimeout(() => {
              new_paste_import_btn.removeClass("mdui-color-red-accent");
            }, 600);
          });
      });

      paste_load.on("click", function () {
        if (!paste_file) {
          file_input.val("");
          file_input.get(0).click();
        } else {
          switch_to_text_paste();
        }
      });

      file_paste.on("click", function () {
        file_input.val("");
        file_input.get(0).click();
      });

      file_input.on("change", function (e) {
        set_paste_file(e.target.files[0]);
      });

      text_input.on("dragover", function (e) {
        e.preventDefault();
        e.stopPropagation();
      });

      drop_file_overlay.on("dragover", function (e) {
        e.preventDefault();
        e.stopPropagation();
      });

      file_paste.on("dragover", function (e) {
        e.preventDefault();
        e.stopPropagation();
      });

      text_input.on("dragenter", function (e) {
        drop_file_overlay.removeClass("mdui-hidden");
      });

      function hide_drop_file_overlay(e) {
        drop_file_overlay.addClass("mdui-hidden");
      }
      drop_file_overlay.on("dragleave", hide_drop_file_overlay);

      function drop_file(e) {
        e.preventDefault();
        e.stopPropagation();
        set_paste_file(e.dataTransfer.files[0]);
        hide_drop_file_overlay(e);
      }

      text_input.on("drop", drop_file);
      drop_file_overlay.on("drop", drop_file);
      file_paste.on("drop", drop_file);

      function check_uuid(uuid) {
        return uuid.length == 0 || /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/im.test(uuid);
      }

      paste_uuid.on("input", function () {
        let uuid = paste_uuid.val();
        let uuid_valid = check_uuid(uuid);
        if (!uuid_valid) {
          paste_uuid.closest(".mdui-textfield").addClass("mdui-textfield-invalid");
        } else {
          paste_uuid.closest(".mdui-textfield").removeClass("mdui-textfield-invalid");
        }
        if (uuid_valid && uuid.length != 0) {
          paste_delete.removeAttr("disabled");
          paste_update.removeAttr("disabled");
          new_paste_import_btn.show();
        } else {
          paste_delete.attr("disabled", "disabled");
          paste_update.attr("disabled", "disabled");
          new_paste_import_btn.hide();
        }
      });

      function check_short_url(url) {
        return url.length == 0 || /^[A-Za-z0-9\\._-]+$/im.test(url);
      }

      const check_short_url_available = _.debounce(function (url) {
        if (url.length == 0) {
          return;
        }
        $.ajax({
          method: "GET",
          url: "api/paste/check_shorturl/" + url
        }).then(res => {
          response = JSON.parse(res);
          if (!response.available) {
            short_url_error.text("此短链接已被占用");
            paste_short_url.closest(".mdui-textfield").addClass("mdui-textfield-invalid");
          }
        });
      }, 300);

      paste_short_url.on("input", function () {
        let url = paste_short_url.val();
        if (!check_short_url(url)) {
          short_url_error.text("仅可使用大小写字母(A-Za-z) 数字(0-9) 部分特殊符号[连字符(-) 下划线(_) 点(.)]");
          paste_short_url.closest(".mdui-textfield").addClass("mdui-textfield-invalid");
        } else {
          paste_short_url.closest(".mdui-textfield").removeClass("mdui-textfield-invalid");
          check_short_url_available(url);
        }
      });

      function check_allow_delete_if_not_available() {
        let max_access_count = paste_max_access_count.val();
        if (paste_expire.val() != "0" || (max_access_count.length && !isNaN(parseInt(max_access_count)))) {
          paste_delete_if_not_available.removeAttr("disabled");
        } else {
          paste_delete_if_not_available.attr("disabled", "disabled");
          paste_delete_if_not_available.prop("checked", false);
        }
      }

      paste_expire.on("change", check_allow_delete_if_not_available);
      paste_max_access_count.on("input", check_allow_delete_if_not_available);

      const result_collapse = new Collapse(new_paste_result);

      function show_result(title, response, sort) {
        new_paste_result_title.text(title);
        let raw_html = "";
        let msgs = Object.entries(response);
        if (sort) {
          msgs = msgs.sort((a, b) => (a[0] + a[1]).length - (b[0] + b[1]).length);
        }
        for (let [k, v] of msgs) {
          if (k == "code" || k == "url") {
            continue;
          }
          if (raw_html.length != 0) {
            raw_html += "\n";
          }
          raw_html += `<p><strong>${k}:</strong> ${v}</p>`;
        }
        new_paste_result_raw.html(raw_html);
        if (response.url) {
          new_paste_result_link.attr("href", response.url).text(response.url);
          if (isDesktop()) {
            new_paste_result_link.attr("target", "_blank");
          }
          QRCode.toCanvas(new_paste_result_qr_code.get(0), response.url, { margin: 0, scale: 6, color: { light: "#00000000", dark: "#000000ff" } }, function () {});
          new_paste_result_link.closest(".mdui-card").find(".paste-link").show();
          new_paste_result_qr_code.show();
        } else {
          new_paste_result_link.closest(".mdui-card").find(".paste-link").hide();
          new_paste_result_qr_code.hide();
        }
        if (response.uuid) {
          paste_uuid.val(response.uuid);
        }
        paste_uuid.get(0).dispatchEvent(new Event("input"));
        return new Promise(resolve => {
          let from = document.documentElement.scrollTop;
          let to = 0;
          if (from > to) {
            let start_time = new Date().getTime();
            (function animate(duration) {
              let progress = Math.max(Math.min((new Date().getTime() - start_time) / duration, 1), 0);
              document.documentElement.scrollTop = from - (from - to) * easeInSine(progress);
              if (progress < 1) {
                requestAnimationFrame(() => animate(duration));
              } else {
                document.documentElement.scrollTop = to;
                resolve();
              }
            })(300);
          } else {
            resolve();
          }
        }).then(() => {
          result_collapse.open();
        });
      }

      function hide_result() {
        return result_collapse.close();
      }

      let upload_progress = (function () {
        let base_loaded = 0;
        let loaded_size = 0;
        let total_size = 0;
        let easeDuration = 1500;
        let last_update = 0;
        let animation_flag = false;

        let complete_promise;
        let complete;

        function open_progress() {
          file_paste_progress_text.text("0.00 MiB / " + (Math.ceil((total_size / 1024 / 1024) * 100) / 100).toFixed(2) + " MiB - 0.00%");
          file_paste_progress_bar.css("width", "0%");
          collapse_file_paste_progress.open();
        }

        function close_progress() {
          collapse_file_paste_progress.close();
        }

        function init_progress() {
          complete_promise = new Promise(resolve => {
            complete = function () {
              close_progress();
              init_progress();
              resolve();
            };
          });
          base_loaded = 0;
          loaded_size = 0;
          total_size = 0;
          easeDuration = 1500;
          last_update = 0;
          animation_flag = false;
        }

        function easeOutSine(x) {
          return Math.sin((x * Math.PI) / 2);
        }

        function now_loaded() {
          let fake_loaded = base_loaded + easeOutSine(Math.min((new Date().getTime() - last_update) / easeDuration, 1)) * (loaded_size - base_loaded);
          return fake_loaded;
        }

        function update_progress() {
          let loaded = now_loaded();
          file_paste_progress_text.text(
            (Math.ceil((loaded / 1024 / 1024) * 100) / 100).toFixed(2) +
              " MiB / " +
              (Math.ceil((total_size / 1024 / 1024) * 100) / 100).toFixed(2) +
              " MiB - " +
              (Math.min(loaded / total_size, 1) * 100).toFixed(2) +
              "%"
          );
          file_paste_progress_bar.css("width", (Math.min(loaded / total_size, 1) * 100).toFixed(2) + "%");
          if (Math.round(loaded) < total_size) {
            requestAnimationFrame(update_progress);
          } else {
            file_paste_progress_text.text((total_size / 1024 / 1024).toFixed(2) + " MiB / " + (total_size / 1024 / 1024).toFixed(2) + " MiB - 100.00%");
            file_paste_progress_bar.css("width", "100%");
            complete();
          }
        }

        function set_progress(e) {
          if (e.lengthComputable) {
            if (e.loaded == 0) {
              open_progress();
            }
            base_loaded = now_loaded();
            if (e.loaded > loaded_size) {
              loaded_size = e.loaded;
            }
            total_size = e.total;

            last_update = new Date().getTime();
            if (!animation_flag) {
              animation_flag = true;
              requestAnimationFrame(update_progress);
            }
          }
          return complete_promise;
        }
        init_progress();
        return set_progress;
      })();

      function sanitizeFilename(filename) {
        return filename.replace(/[\/\\:*?"<>|]+/g, "_");
      }

      function prepare_data() {
        let text = text_input.val();
        let password = paste_password.val();
        let expire = paste_expire.val();
        let max_access_count = paste_max_access_count.val();
        let short_url = paste_short_url.val();
        let delete_if_not_available = paste_delete_if_not_available.prop("checked");
        let detect_mime = paste_detect_mime.prop("checked");
        let shorten_url = paste_shorten_url.prop("checked");
        let data = new FormData();
        let query_params = {};
        if (password.length != 0) {
          query_params.password = password;
        }

        if (expire != "0") {
          query_params.expire_after = new Date().getTime() + parseInt(expire) * 1000;
        }

        if (max_access_count.length != 0) {
          if (!/^[\d]+$/m.test(max_access_count) || isNaN(parseInt(max_access_count))) {
            mdui.snackbar("最大访问次数必须为数字");
            return;
          }
          query_params.max_access_count = parseInt(max_access_count, 10);
        }

        if (short_url.length != 0) {
          if (paste_short_url.closest(".mdui-textfield").hasClass("mdui-textfield-invalid")) {
            mdui.snackbar(short_url_error.text());
            return;
          }
          query_params.short_url = short_url;
        }
        if (paste_delete_if_not_available.attr("disabled") != "disabled") {
          if (delete_if_not_available) {
            query_params.delete_if_not_available = "true";
          } else {
            delete query_params.delete_if_not_available;
          }
        }

        if (paste_file) {
          if (detect_mime) {
            data.append("c", new File([paste_file], paste_file.name, { type: "application/vnd.pastebin.detect" }));
          } else {
            data.append("c", paste_file);
          }
        } else {
          let filename = sanitizeFilename(text || "-").substring(0, 36);
          if (detect_mime) {
            data.append("c", new File([text], filename, { type: "application/vnd.pastebin.detect" }));
          } else if (shorten_url) {
            data.append("c", new File([text], filename, { type: "application/vnd.pastebin.shorten" }));
          } else {
            data.append("c", new File([text], filename + ".txt", { type: "text/plain; charset=utf-8" }));
          }
        }
        return { data, query_params };
      }

      paste_submit.on("click", async function () {
        let prepared_data = prepare_data();
        let data = prepared_data.data;
        let query_params = prepared_data.query_params;
        action_button.addClass("action-disabled");
        await hide_result();
        const query_string = $.param(query_params).trim();
        $.ajax({
          method: "POST",
          url: query_string != "" ? "?" + query_string : "",
          data: data,
          headers: {
            Accept: "application/json",
            "X-Paste-Size": (data.get("c") || { size: 0 }).size
          },
          contentType: false,
          processData: false,
          beforeSend: function (xhr) {
            if (paste_file) {
              upload_progress({ loaded: 0, total: paste_file.size, lengthComputable: true });
              xhr.upload.addEventListener("progress", upload_progress);
            }
          },
          complete: async function (xhr) {
            let response = JSON.parse(xhr.responseText || "{}");
            if (xhr.responseText == "" || !response || response.code != 0) {
              paste_submit.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
              setTimeout(() => {
                paste_submit.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
              }, 600);
              if (!response.error) {
                response.error = "network error";
              }
              show_result("创建失败", response, false);
            } else {
              if (paste_file) {
                await upload_progress({ loaded: paste_file.size, total: paste_file.size, lengthComputable: true });
              }
              paste_submit.removeClass("mdui-color-theme-accent").addClass("mdui-color-green-600");
              setTimeout(() => {
                paste_submit.removeClass("mdui-color-green-600").addClass("mdui-color-theme-accent");
              }, 600);
              show_result("创建结果", response, false);
            }
            action_button.removeClass("action-disabled");
          }
        });
      });

      function update_paste() {
        return new Promise(async (resolve, reject) => {
          let uuid = paste_uuid.val();
          if (!check_uuid(uuid) || uuid.length == 0) {
            mdui.snackbar("无效的 UUID");
            return reject({ error: "无效的 UUID" });
          }
          let prepared_data = prepare_data();
          let data = prepared_data.data;
          if (data.get("c").size == 0) {
            data.delete("c");
          }
          let query_params = prepared_data.query_params;

          action_button.addClass("action-disabled");
          await hide_result();
          const query_string = $.param(query_params).trim();
          $.ajax({
            method: "PUT",
            url: uuid + (query_string != "" ? "?" + query_string : ""),
            data: data,
            headers: {
              Accept: "application/json",
              "X-Paste-Size": (data.get("c") || { size: 0 }).size
            },
            contentType: false,
            processData: false,
            beforeSend: function (xhr) {
              if (paste_file) {
                upload_progress({ loaded: 0, total: paste_file.size, lengthComputable: true });
                xhr.upload.addEventListener("progress", upload_progress);
              }
            },
            complete: async function (xhr) {
              let response = JSON.parse(xhr.responseText || "{}");
              if (xhr.responseText == "" || !response || response.code != 0) {
                paste_update.removeClass("mdui-color-blue-accent").addClass("mdui-color-red-accent");
                setTimeout(() => {
                  paste_update.removeClass("mdui-color-red-accent").addClass("mdui-color-blue-accent");
                }, 600);
                if (!response.error) {
                  response.error = "network error";
                }
                reject(response);
              } else {
                if (paste_file) {
                  await upload_progress({ loaded: paste_file.size, total: paste_file.size, lengthComputable: true });
                }
                paste_update.removeClass("mdui-color-blue-accent").addClass("mdui-color-green-600");
                setTimeout(() => {
                  paste_update.removeClass("mdui-color-green-600").addClass("mdui-color-blue-accent");
                }, 600);
                resolve(response);
              }
              action_button.removeClass("action-disabled");
            }
          });
        });
      }

      paste_float_update.on("click", function () {
        paste_float_update.addClass("uploading");
        update_paste()
          .then(response => {
            paste_float_update.removeClass("mdui-color-theme-accent").addClass("mdui-color-green-600");
            setTimeout(() => {
              paste_float_update.removeClass("mdui-color-green-600").addClass("mdui-color-theme-accent");
            }, 1500);
          })
          .catch(response => {
            paste_float_update.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
            setTimeout(() => {
              paste_float_update.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
            }, 1500);
            mdui.snackbar("更新失败：" + response.error || "未知错误");
          })
          .finally(() => {
            paste_float_update.removeClass("uploading");
          });
      });

      let showFloatUpdate = _.throttle(function () {
        if (paste_update.get(0).disabled) {
          if (!paste_float_update.hasClass("mdui-fab-hide")) {
            paste_float_update.addClass("mdui-fab-hide");
          }
          return;
        }
        let update_location = paste_update.offset().top;
        if (document.documentElement.scrollTop + document.documentElement.clientHeight < update_location) {
          paste_float_update.removeClass("mdui-fab-hide");
        } else {
          paste_float_update.addClass("mdui-fab-hide");
        }
      }, 100);

      $(window).on("scroll", function () {
        showFloatUpdate();
        return true;
      });

      text_input.on("input", function () {
        showFloatUpdate();
      });

      paste_update.on("click", () => {
        update_paste()
          .then(response => {
            show_result("更新结果", response, false);
          })
          .catch(response => {
            show_result("更新失败", response, false);
          });
      });

      async function delete_paste(force) {
        let uuid = paste_uuid.val();
        if (!check_uuid(uuid) || uuid.length == 0) {
          mdui.snackbar("无效的 UUID");
          return;
        }
        action_button.addClass("action-disabled");
        await hide_result();
        $.ajax({
          method: "DELETE",
          url: uuid + (force ? "?force=true" : ""),
          headers: {
            Accept: "application/json"
          },
          contentType: false,
          processData: false,
          complete: function (xhr) {
            let response = JSON.parse(xhr.responseText || "{}");
            if (xhr.responseText == "" || !response || response.code != 0) {
              paste_delete.removeClass("mdui-color-red").addClass("mdui-color-red-800");
              setTimeout(() => {
                paste_delete.removeClass("mdui-color-red-800").addClass("mdui-color-red");
              }, 600);
              if (!response.error) {
                response.error = "network error";
              }
              show_result("删除失败", response, true);
            } else {
              paste_delete.removeClass("mdui-color-red").addClass("mdui-color-green-600");
              setTimeout(() => {
                paste_delete.removeClass("mdui-color-green-600").addClass("mdui-color-red");
              }, 600);
              show_result("删除结果" + (force ? "：强制删除" : ""), response, true);
            }
            action_button.removeClass("action-disabled");
          }
        });
      }

      paste_delete.on("click", function (e) {
        let force_delete = e.shiftKey || paste_force_delete;
        if (paste_force_delete) {
          paste_force_delete = false;
        }
        delete_paste(force_delete);
      });

      let delete_torch_start = 0;
      const delete_torch_long_press_threshold = 1000;
      paste_delete.on("touchstart", function (e) {
        e.preventDefault();
        delete_torch_start = new Date().getTime();
      });

      paste_delete.on("touchend", function (e) {
        e.preventDefault();
        let duration = new Date().getTime() - delete_torch_start;
        if (duration >= delete_torch_long_press_threshold) {
          delete_paste(true);
        } else {
          delete_paste(false);
        }
      });

      new_paste_result_copy.on("click", function () {
        function selectAndHint() {
          let selection = window.getSelection();
          let range = document.createRange();
          range.selectNodeContents(new_paste_result_link.get(0));
          selection.removeAllRanges();
          selection.addRange(range);
          mdui.snackbar("请按 Ctrl+C 复制");
        }
        if (navigator.clipboard) {
          navigator.clipboard
            .writeText(new_paste_result_link.text())
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
      (async function edit_mode() {
        let query_params = new URLSearchParams(location.search);
        let uuid = query_params.get("edit");
        if (uuid && check_uuid(uuid)) {
          if (!config.allow_anonymous) {
            if (!(await user_is_login)) return;
          }
          paste_uuid.val(uuid);
          paste_uuid.get(0).dispatchEvent(new Event("input"));
          paste_uuid.attr("disabled", "disabled");
          new_paste_delete_container.hide();
          new_paste_submit_container.hide();
          new_paste_update_container.removeClass("mdui-col-md-4");
          new_paste_return_container.show();
          new_paste_return.on("click", function () {
            history.back();
          });
          paste_viewer_tab.addClass("mdui-hidden");
          paste_manage_tab.addClass("mdui-hidden");
          paste_app_tab.show(0);
        }
      })();
    })();
    let paste_viewer_collapses;
    (function paste_viewer() {
      const paste_viewer_query = $("#paste-viewer-query");
      const paste_viewer_password = $("#paste-viewer-password");
      const paste_viewer_password_form = $("#paste-viewer-password-form");
      const paste_viewer_text = $("#paste-viewer-text");

      const paste_viewer_not_found = $("#paste-viewer-not-found");
      const paste_viewer_file = $("#paste-viewer-file");
      paste_viewer_collapses = new CollapseGroup({
        paste_viewer_query,
        paste_viewer_password,
        paste_viewer_not_found,
        paste_viewer_text,
        paste_viewer_file
      });

      const paste_viewer_back_to_query = $(".paste-viewer-back-to-query");
      const paste_viewer_text_copy = $("#paste-viewer-text-copy");
      const paste_viewer_action = $(".paste-viewer-action");
      const paste_viewer_query_form = $("#paste-viewer-query-form");
      const paste_viewer_query_input = $("#paste-viewer-query-input");
      const paste_viewer_progress = $(".paste-viewer-progress");

      const paste_viewer_title = $(".paste-viewer-title");
      const paste_viewer_password_input = $("#paste-viewer-password-input");
      const paste_viewer_confirm_password = $("#paste-viewer-confirm-password");

      const paste_viewer_text_content = $("#paste-viewer-text-content");
      const paste_viewer_enable_markdown_render = $("#paste-viewer-enable-markdown-render");

      const paste_viewer_download_btn = $(".paste-viewer-download-btn");
      const paste_viewer_file_icon = $("#paste-viewer-file-icon");
      const paste_viewer_file_preview = $("#paste-viewer-file-preview");
      const paste_viewer_file_filename = $("#paste-viewer-file-filename");

      const paste_viewer_enable_highlight_js = $("#paste-viewer-enable-highlight-js");
      const paste_viewer_highlight_language = $("#paste-viewer-highlight-language");
      const paste_viewer_highlight_language_selector = (function initHighlightLanguage() {
        let languages = hljs.listLanguages();
        for (let lang of languages) {
          paste_viewer_highlight_language.append($(`<option value="${lang}">${lang}</option>`));
        }
        paste_viewer_highlight_language.get(0).selectedIndex = -1;
        return new mdui.Select(paste_viewer_highlight_language.get(0));
      })();
      let viewer_mode = false;

      let query_id;
      let paste_metadata = {
        id: "",
        size: 9 * 1024 * 1024,
        type: "image/png",
        filename: "Cirno.png",
        access_token: "",
        content: "",
        url: ""
      };
      function paste_preview_file_show() {
        function show_preview() {
          paste_viewer_collapses.paste_viewer_file.open();
          action_unlock();
        }
        let timeout = setTimeout(() => {
          timeout = null;
          show_preview();
        }, 600);
        let start = new Date().getTime();
        return function show() {
          if (timeout) {
            clearTimeout(timeout);
            let end = new Date().getTime();
            if (end - start < 300) {
              setTimeout(
                () => {
                  show_preview();
                },
                300 - (end - start)
              );
            } else {
              show_preview();
            }
          }
        };
      }
      let paste_viewer_file_preview_element;

      function paste_preview_file() {
        paste_viewer_file_filename.text(paste_metadata.filename + " (" + (paste_metadata.size / 1024 / 1024).toFixed(2).toString() + " MiB)");
        paste_viewer_file_icon.hide();
        paste_viewer_file_preview.show();
        if (paste_viewer_file_preview_element) {
          paste_viewer_file_preview_element.remove();
          paste_preview_file_preview_element = null;
        }
        let show = paste_preview_file_show();
        if (paste_metadata.type.startsWith("image/")) {
          paste_viewer_file_preview_element = $(`<img style="max-height: inherit; max-width:100%">`)
            .on("load", show)
            .attr("src", paste_metadata.url)
            .appendTo(paste_viewer_file_preview);
        } else if (paste_metadata.type.startsWith("audio/")) {
          paste_viewer_file_preview_element = $('<audio controls style="max-height: inherit; max-width:100%">')
            .on("loadedmetadata", show)
            .attr("src", paste_metadata.url)
            .appendTo(paste_viewer_file_preview);
        } else if (paste_metadata.type.startsWith("video/")) {
          paste_viewer_file_preview_element = $('<video controls style="max-height: inherit; max-width:100%">')
            .on("loadedmetadata", show)
            .attr("src", paste_metadata.url)
            .appendTo(paste_viewer_file_preview);
        } else {
          paste_viewer_file_preview.hide();
          paste_viewer_file_icon.show();
        }
      }

      function paste_preview_text_render(init) {
        if (!paste_viewer_enable_highlight_js.prop("checked")) {
          paste_viewer_highlight_language.closest(".mdui-row").hide();
        }
        if (paste_viewer_enable_markdown_render.prop("checked")) {
          paste_viewer_text_content.css("white-space", "");
          paste_viewer_text_content.html(DOMPurify.sanitize(marked.parse(paste_metadata.content)));
        } else if (paste_viewer_enable_highlight_js.prop("checked")) {
          paste_viewer_text_content.css("white-space", "pre");
          if (!paste_viewer_highlight_language.val()) {
            let highlighted = hljs.highlightAuto(paste_metadata.content);
            paste_viewer_highlight_language.val(highlighted.language);
            paste_viewer_highlight_language_selector.handleUpdate();
            paste_viewer_text_content.html(hljs.lineNumbersValue(highlighted.value));
          } else {
            paste_viewer_text_content.html(hljs.lineNumbersValue(hljs.highlight(paste_metadata.content, { language: paste_viewer_highlight_language.val() }).value));
          }
          paste_viewer_highlight_language.closest(".mdui-row").show();
        } else {
          paste_viewer_text_content.css("white-space", "pre-wrap");
          paste_viewer_text_content.text(paste_metadata.content);
        }
        paste_viewer_collapses.paste_viewer_text.open();
      }

      paste_viewer_enable_markdown_render.on("change", () => {
        if (paste_viewer_enable_markdown_render.prop("checked")) {
          paste_viewer_enable_highlight_js.prop("checked", false);
        }
        paste_preview_text_render();
      });

      paste_viewer_enable_highlight_js.on("change", () => {
        if (paste_viewer_enable_highlight_js.prop("checked")) {
          paste_viewer_enable_markdown_render.prop("checked", false);
        }
        paste_preview_text_render();
      });

      paste_viewer_highlight_language.on("change", () => {
        paste_preview_text_render();
      });

      function paste_preview_text() {
        $.ajax({
          method: "GET",
          url: paste_metadata.id + (paste_metadata.type == "application/vnd.pastebin.shorten" ? "/raw" : "") //+ "?access_token=" + paste_metadata.access_token,
        })
          .then(res => {
            paste_metadata.content = res;
            if (paste_metadata.type.startsWith("text/markdown") || paste_metadata.type == "application/vnd.pastebin.shorten") {
              paste_viewer_enable_markdown_render.prop("checked", true);
            }

            paste_preview_text_render(true);
            action_unlock();
          })
          .catch(() => {
            paste_viewer_text_content.text("无法加载 Paste");
            action_unlock();
          });
      }

      function paste_preview() {
        if ((paste_metadata.type.startsWith("text/") || paste_metadata.type == "application/vnd.pastebin.shorten") && paste_metadata.size <= 1024 * 1024) {
          paste_preview_text();
        } else {
          paste_preview_file();
        }
      }

      function parse_filename(xhr) {
        let urlencode_filename = xhr.getResponseHeader("X-Origin-Filename-Encoded");
        return decodeURIComponent(urlencode_filename);
      }

      function action_lock() {
        paste_viewer_action.attr("disabled", "disabled");
        paste_viewer_progress.show();
      }
      function action_unlock() {
        paste_viewer_action.removeAttr("disabled");
        paste_viewer_progress.hide();
      }
      function query_paste_metadata(id, password) {
        action_lock();
        paste_viewer_title.text("Paste: " + id);
        query_id = id;
        $.ajax({
          method: "HEAD",
          url: id + (password ? "?pwd=" + password : ""),
          complete: function (xhr) {
            if (xhr.status == 200) {
              paste_metadata = {
                id: id,
                password: password,
                size: xhr.getResponseHeader("Content-Length"),
                type: xhr.getResponseHeader("Content-Type"),
                filename: parse_filename(xhr),
                access_token: xhr.getResponseHeader("X-Access-Token"),
                url: id + "?access_token=" + xhr.getResponseHeader("X-Access-Token")
              };
              paste_viewer_download_btn.attr("download", paste_metadata.filename).attr("href", paste_metadata.url);
              paste_preview();
            } else if (xhr.status == 404) {
              paste_viewer_collapses.paste_viewer_not_found.open();
              action_unlock();
            } else if (xhr.status == 401) {
              if (!password) {
                paste_viewer_collapses.paste_viewer_password.open();
              } else {
                paste_viewer_confirm_password.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
                setTimeout(() => {
                  paste_viewer_confirm_password.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
                }, 600);
                mdui.snackbar("密码错误");
              }
              action_unlock();
            } else {
              action_unlock();
            }
          }
        });
      }

      paste_viewer_query_form.on("submit", function () {
        query_id = paste_viewer_query_input.val();
        if (query_id.length == 0) {
          mdui.snackbar("请输入 Paste Short URL 或 Paste Hash");
          return;
        }
        query_id = query_id.split("/").pop().replace(/^#/m, "");
        query_paste_metadata(query_id);
      });

      paste_viewer_back_to_query.on("click", function () {
        paste_viewer_collapses.paste_viewer_query.open().then(() => {
          paste_viewer_back_to_query.trigger("pastebin.viewer.clean");
        });
        if (viewer_mode) {
          history.back();
        }
      });

      paste_viewer_text_copy.on("click", function () {
        let text = $("#paste-viewer-text-content");
        function selectAndHint() {
          let selection = window.getSelection();
          let range = document.createRange();
          range.selectNodeContents(text.get(0));
          selection.removeAllRanges();
          selection.addRange(range);
          mdui.snackbar("请按 Ctrl+C 复制");
        }
        if (navigator.clipboard) {
          navigator.clipboard
            .writeText(text.text())
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

      paste_viewer_back_to_query.on("pastebin.viewer.clean", function () {
        if (paste_viewer_file_preview_element) {
          paste_viewer_file_preview_element.remove();
          paste_viewer_file_preview_element = null;
        }
      });

      paste_viewer_password_form.on("submit", function () {
        let password = paste_viewer_password_input.val();
        if (password.length == 0) {
          mdui.snackbar("请输入密码");
          return;
        }
        query_paste_metadata(query_id, password);
      });

      if (query_hash) {
        let query_params = new URLSearchParams(location.search);
        let password = query_params.get("pwd");
        let markdown_enable = query_params.has("md");
        let highlight_enable = query_params.has("hl");
        let highlight_language = query_params.get("hl");
        viewer_mode = query_params.has("viewer_mode") && query_params.get("viewer_mode") == "true";
        if (password) {
          paste_viewer_password_input.val(password);
          paste_viewer_password_input.get(0).dispatchEvent(new Event("input"));
        }
        if (markdown_enable && !highlight_enable) {
          paste_viewer_enable_markdown_render.prop("checked", true);
        }
        if (highlight_enable) {
          paste_viewer_enable_highlight_js.prop("checked", true);
          if (highlight_language) {
            paste_viewer_highlight_language.val(highlight_language);
            paste_viewer_highlight_language_selector.handleUpdate();
          }
        }
        if (viewer_mode) {
          new_paste_tab.addClass("mdui-hidden");
          paste_manage_tab.addClass("mdui-hidden");
        }
        paste_app_tab.show(1);
        paste_viewer_query_input.val(query_hash);
        paste_viewer_query_input.get(0).dispatchEvent(new Event("input"));
        query_paste_metadata(query_hash);
      }
    })();
    (function paste_manage() {
      const new_paste_uuid = $("#new-paste-uuid");
      const new_paste_delete = $("#new-paste-delete").get(0);

      const paste_viewer_query_input = $("#paste-viewer-query-input");
      const paste_viewer_progress = $(".paste-viewer-progress");
      const paste_viewer_query_btn = $("#paste-viewer-query-btn");
      let paste_viewer_back_to_manage = false;
      let paste_manger_store_scroll_top = 0;
      const paste_viewer_back_to_query = $(".paste-viewer-back-to-query");

      const paste_manage_pastes = $("#paste-manage-pastes");
      const paste_manage_null = $("#paste-manage-null");

      const paste_manage_progress = $(".paste-manage-progress");
      const paste_manage_refresh = $("#paste-manage-refresh");
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

        paste_manage_view_btn.on("click", async function (e) {
          paste_viewer_query_input.val(hash);
          paste_viewer_query_input.get(0).dispatchEvent(new Event("input"));
          paste_viewer_progress.show();
          paste_viewer_query_btn.attr("disabled", "disabled");
          paste_manger_store_scroll_top = document.documentElement.scrollTop;
          paste_app_tab.show(1);
          await paste_viewer_collapses.paste_viewer_query.open();
          paste_viewer_back_to_manage = true;
          $.ajax({
            method: "GET",
            url: "api/paste/" + uuid,
            headers: {
              Accept: "application/json"
            },
            complete: function (xhr) {
              let response = JSON.parse(xhr.responseText || "{}");
              if (xhr.status == 200 && response.code === 0) {
                paste_viewer_query_btn.removeAttr("disabled");
                paste_viewer_query_btn.get(0).click();
              } else {
                mdui.snackbar("加载失败: " + response.error);
                paste_app_tab.show(2);
              }
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
          new_paste_uuid.val(uuid);
          new_paste_uuid.get(0).dispatchEvent(new Event("input"));
          paste_app_tab.show(0);
          paste_force_delete = true;
          new_paste_delete.click();
        });

        paste_manage_edit_btn.on("click", function (e) {
          new_paste_uuid.val(uuid);
          new_paste_uuid.get(0).dispatchEvent(new Event("input"));
          paste_app_tab.show(0);
        });
      }

      function generate_paste_panel_html(paste) {
        let now = new Date().getTime();
        let expired = new Date(paste.expire_after).getTime();
        let pastes_panel = `
          <div class="mdui-panel-item${paste_detail_opened.has(paste.uuid) ? " mdui-panel-item-open" : ""}">
            <div class="mdui-panel-item-header">
        `;
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
          pastes_panel += `<div class="mdui-panel-item-title" style="flex-grow: 1">${paste.filename}</div>`;
        } else {
          pastes_panel += `<div class="mdui-panel-item-title" style="flex-grow: 1">${paste.short_url || paste.hash}</div>`;
        }
        pastes_panel += `
          <div class="mdui-panel-item-summary mdui-hidden-xs-down">Time: ${paste.created_at.substring(0, Math.min(23, paste.created_at.length))}</div>
        `;
        pastes_panel += `
              <i class="mdui-panel-item-arrow mdui-icon material-icons">keyboard_arrow_down</i>
            </div>
            <div class="mdui-panel-item-body">
              <div class="raw-result">
                <button class="mdui-btn mdui-btn-icon mdui-ripple paste-manage-copy-url-btn copy-btn mdui-float-right">
                  <i class="mdui-icon material-icons">content_copy</i>
                </button>
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
        paste_manage_refresh.addClass("refreshing");
        paste_manage_refresh.attr("disabled", "disabled");
        paste_manage_pager.attr("disabled", "disabled");
        $.ajax({
          method: "GET",
          url: "api/user/pastes",
          headers: {
            Accept: "application/json"
          },
          data: {
            page: page,
            page_size: page_size
          },
          complete: function (xhr) {
            let response = JSON.parse(xhr.responseText || "{}");
            if (xhr.status == 200 && response.code === 0) {
              paste_total = response.total;
              max_page = Math.ceil(paste_total / page_size);
              if (page > max_page) {
                page = max_page;
                return list_paste(scrollOffset);
              }
              if (response.pastes.length != 0) {
                let paste_map = new Map();
                for (let paste of response.pastes) {
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
                      start_node = panel;
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
            paste_manage_refresh.removeClass("refreshing");
            paste_manage_refresh.removeAttr("disabled");
            if (scrollOffset) {
              window.scrollTo(0, document.documentElement.scrollHeight - scrollOffset);
            }
          }
        });
      }

      paste_manage_refresh.on("click", function () {
        updateUserPasteSize();
        list_paste(document.documentElement.scrollHeight - document.documentElement.scrollTop);
      });

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

      paste_app_tab_element.on("change.mdui.tab", function (e) {
        if (e.detail.index == 2) {
          if (!paste_viewer_back_to_manage) {
            paste_manage_pastes.hide();
            paste_manage_null.show();
          }
          list_paste();
          paste_viewer_back_to_manage = false;
        }
      });

      paste_manage_tab.on("show.mdui.tab", function () {
        updateUserPasteSize();
        if (paste_manger_store_scroll_top) {
          (function (target) {
            requestAnimationFrame(() => {
              window.scrollTo(0, target);
            });
          })(paste_manger_store_scroll_top);
          paste_manger_store_scroll_top = 0;
        }
      });
    })();
    (function user_profile() {
      const account_dialog_btn = $("#account-dialog-btn");
      const user_profile_view = $(".user-profile-view");
      const user_profile_edit = $(".user-profile-edit");
      const user_profile_dialog = new mdui.Dialog("#user-profile-dialog", { history: false });
      const user_profile_dialog_progress = $(".user-profile-dialog-progress");

      const user_profile_uid_text = $("#user-profile-uid-text");
      const user_profile_username_text = $("#user-profile-username-text");
      const user_profile_role_text = $("#user-profile-role-text");
      const user_profile_email_text = $("#user-profile-email-text");
      const user_profile_manage_panel = $("#user-profile-manage-panel");
      const user_profile_edit_btn = $("#user-profile-edit-btn");
      const user_profile_webauthn_manage_btn = $("#user-profile-webauthn-manage-btn");

      const user_profile_edit_username = $("#user-profile-edit-username");
      const user_profile_edit_email = $("#user-profile-edit-email");
      const user_profile_edit_oldpwd = $("#user-profile-edit-oldpwd");
      const user_profile_edit_newpwd = $("#user-profile-edit-newpwd");
      const user_profile_edit_action = $(".user-profile-edit-action");
      const user_profile_edit_confirm = $("#user-profile-edit-confirm");
      const user_profile_edit_return = $("#user-profile-edit-return");

      const user_profile_webauthn_manage = $(".user-profile-webauthn-manage");
      const user_profile_webauthn_manage_list = $("#user-profile-webauthn-manage-list");
      const user_profile_webauthn_manage_add = $("#user-profile-webauthn-manage-add");
      const user_profile_webauthn_manage_return = $("#user-profile-webauthn-manage-return");
      const user_profile_webauthn_manage_main = $("#user-profile-webauthn-manage-main");
      const user_profile_webauthn_manage_name = $("#user-profile-webauthn-manage-name");
      const user_profile_webauthn_manage_name_input = $("#user-profile-webauthn-manage-name-input");
      const user_profile_webauthn_manage_register_as_passkey = $("#user-profile-webauthn-manage-register-as-passkey");
      const user_profile_webauthn_manage_name_cancel = $("#user-profile-webauthn-manage-name-cancel");
      const user_profile_webauthn_manage_name_confirm = $("#user-profile-webauthn-manage-name-confirm");
      const user_profile_webauthn_manage_name_row = $("#user-profile-webauthn-manage-name-row");
      const user_profile_webauthn_manage_name_loading = $("#user-profile-webauthn-manage-name-loading");
      const user_profile_webauthn_manage_action = $(".user-profile-webauthn-manage-action");

      const login_dialog = new mdui.Dialog("#user-login-dialog", { history: false });
      const login_button = $("#user-login-button");
      const login_form = $("#user-login-form");
      const login_username = $("#user-login-username");
      const login_password = $("#user-login-password");
      const login_dialog_action = $(".user-login-dialog-action");

      const login_webauthn = $("#user-login-webauthn");

      function show_user_profile() {
        if (user_info) {
          user_profile_uid_text.text(user_info.uid);
          user_profile_username_text.text(user_info.username);
          user_profile_role_text.text(user_info.role);
          user_profile_email_text.text(user_info.email);
          if (user_info.role == "admin") {
            user_profile_manage_panel.show();
          } else {
            user_profile_manage_panel.hide();
          }
          user_profile_view.show();
          user_profile_edit.hide();
          user_profile_webauthn_manage.hide();
          user_profile_dialog.handleUpdate();
        } else {
          user_profile_dialog.close();
        }
      }

      function show_user_profile_edit() {
        if (user_info) {
          user_profile_edit_username.attr("placeholder", user_info.username);
          user_profile_edit_email.attr("placeholder", user_info.email);
        }
        user_profile_view.hide();
        user_profile_edit.show();
        user_profile_dialog.handleUpdate();
      }

      user_profile_edit_btn.on("click", function () {
        show_user_profile_edit();
      });

      user_profile_edit_return.on("click", function () {
        show_user_profile();
      });

      let account_button_lock = false;
      account_dialog_btn.on("click", function () {
        if (account_button_lock || !user_is_login) {
          mdui.snackbar("数据加载中，请稍后");
          return;
        }
        account_button_lock = true;
        Promise.race([user_is_login, "loading"]).then(is_login => {
          if (is_login === "loading") {
            mdui.snackbar("数据加载中，请稍后");
          }
        });
        user_is_login.then(is_login => {
          account_button_lock = false;
          if (is_login) {
            show_user_profile();
            user_profile_dialog.open();
          } else {
            login_dialog.open();
          }
        });
      });

      login_form.on("submit", function (e) {
        e.preventDefault();
        let account = login_username.val();
        let password = login_password.val();
        if (account.length == 0 || password.length == 0) {
          mdui.snackbar("用户或密码不能为空");
          return;
        }
        login_dialog_action.attr("disabled", "disabled");
        $.ajax({
          method: "POST",
          url: "api/user/login",
          data: JSON.stringify({
            account: account,
            password: password
          }),
          contentType: "application/json",
          complete: function (xhr) {
            let response = JSON.parse(xhr.responseText || "{}");
            if (xhr.status == 200 && response.code === 0) {
              update_user_info(response.info).then(() => {
                mdui.snackbar("登录成功");
                login_button.removeClass("mdui-color-theme-accent").addClass("mdui-color-green-600");
                setTimeout(() => {
                  login_button.removeClass("mdui-color-green-600").addClass("mdui-color-theme-accent");
                  login_dialog.close();
                }, 600);
                login_dialog_action.removeAttr("disabled");
              });
            } else {
              mdui.snackbar("登录失败: " + response.error);
              login_dialog_action.removeAttr("disabled");
              login_button.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
              setTimeout(() => {
                login_button.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
              }, 600);
            }
          }
        });
      });

      user_profile_edit_confirm.on("click", function () {
        let data = {
          username: user_profile_edit_username.val(),
          email: user_profile_edit_email.val(),
          old_password: user_profile_edit_oldpwd.val(),
          new_password: user_profile_edit_newpwd.val()
        };
        if (data.username.length == 0) {
          delete data.username;
        }
        if (data.email.length == 0) {
          delete data.email;
        }
        if (data.old_password !== undefined && data.old_password.length != 0 && (data.new_password === undefined || data.new_password.length == 0)) {
          mdui.snackbar("请输入新密码");
          return;
        }
        user_profile_edit_action.attr("disabled", "disabled");
        $.ajax({
          method: "POST",
          url: "api/user/edit",
          data: JSON.stringify(data),
          contentType: "application/json",
          complete: function (xhr) {
            let response = JSON.parse(xhr.responseText || "{}");
            if (xhr.status == 200 && response.code === 0) {
              update_user_info().then(is_login => {
                if (is_login) {
                  mdui.snackbar("修改成功");
                } else {
                  mdui.snackbar("修改成功，请重新登录");
                }
                user_profile_edit_confirm.removeClass("mdui-color-theme-accent").addClass("mdui-color-green-600");
                setTimeout(() => {
                  user_profile_edit_confirm.removeClass("mdui-color-green-600").addClass("mdui-color-theme-accent");
                  if (!is_login) {
                    user_profile_dialog.close();
                    login_dialog.open();
                  }
                }, 600);
                user_profile_edit_action.removeAttr("disabled");
              });
            } else {
              mdui.snackbar("修改失败: " + response.error);
              user_profile_edit_confirm.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
              setTimeout(() => {
                user_profile_edit_confirm.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
              }, 600);
              user_profile_edit_action.removeAttr("disabled");
            }
          }
        });
      });

      function bufferEncode(value) {
        return btoa(String.fromCharCode.apply(null, new Uint8Array(value)))
          .replace(/\+/g, "-")
          .replace(/\//g, "_")
          .replace(/=/g, "");
      }

      function bufferDecode(value) {
        return Uint8Array.from(atob(value.replace(/-/g, "+").replace(/_/g, "/")), c => c.charCodeAt(0));
      }

      login_webauthn.on("click", function () {
        let account = login_username.val();
        if (!navigator.credentials) {
          mdui.snackbar("WebAuthn 不可用");
          return;
        }
        let passkey = account.length == 0;
        login_dialog_action.attr("disabled", "disabled");
        $.ajax({
          method: passkey ? "GET" : "POST",
          url: passkey ? "api/user/webauthn/passkey/login/request" : "api/user/webauthn/login/request",
          contentType: "application/json",
          data: passkey
            ? null
            : JSON.stringify({
                account: account
              }),
          processData: false,
          complete: function (xhr) {
            let response = JSON.parse(xhr.responseText || "{}");
            if (xhr.status == 200 && response.code === 0) {
              response.publicKey.challenge = bufferDecode(response.publicKey.challenge);
              if (response.publicKey.allowCredentials && response.publicKey.allowCredentials.length > 0) {
                response.publicKey.allowCredentials = response.publicKey.allowCredentials.map(credential => {
                  credential.id = bufferDecode(credential.id);
                  return credential;
                });
              }
              navigator.credentials
                .get({
                  publicKey: response.publicKey
                })
                .then(credential => {
                  $.ajax({
                    method: "POST",
                    url: passkey ? "api/user/webauthn/passkey/login" : "api/user/webauthn/login",
                    headers: {
                      "X-WebAuthn-Session": response.session
                    },
                    data: JSON.stringify({
                      id: credential.id,
                      rawId: bufferEncode(credential.rawId),
                      type: credential.type,
                      response: {
                        authenticatorData: bufferEncode(credential.response.authenticatorData),
                        clientDataJSON: bufferEncode(credential.response.clientDataJSON),
                        signature: bufferEncode(credential.response.signature),
                        userHandle: bufferEncode(credential.response.userHandle)
                      }
                    }),
                    contentType: "application/json",
                    processData: false,
                    complete: function (xhr) {
                      let response = JSON.parse(xhr.responseText || "{}");
                      if (xhr.status == 200 && response.code === 0) {
                        update_user_info(response.info).then(() => {
                          mdui.snackbar("登录成功");
                          login_button.removeClass("mdui-color-theme-accent").addClass("mdui-color-green-600");
                          setTimeout(() => {
                            login_button.removeClass("mdui-color-green-600").addClass("mdui-color-theme-accent");
                            login_dialog.close();
                          }, 600);
                        });
                      } else {
                        mdui.snackbar("登录失败: " + response.error);
                      }
                      login_dialog_action.removeAttr("disabled");
                    }
                  });
                })
                .catch(err => {
                  mdui.snackbar("登录失败: 身份验证失败，如果你的认证器不是 Passkey 类型，请确保输入了正确的账号");
                  login_dialog_action.removeAttr("disabled");
                });
            } else {
              mdui.snackbar("登录失败: " + response.error);
              login_dialog_action.removeAttr("disabled");
            }
          }
        });
      });

      user_profile_webauthn_manage_btn.on("click", function () {
        user_profile_view.hide();
        user_profile_webauthn_manage.show();
        user_profile_dialog.handleUpdate();
        update_webauthn_list();
      });

      user_profile_webauthn_manage_return.on("click", function () {
        user_profile_webauthn_manage.hide();
        user_profile_view.show();
        user_profile_dialog.handleUpdate();
      });

      user_profile_webauthn_manage_add.on("click", function () {
        user_profile_webauthn_manage_main.hide();
        user_profile_webauthn_manage_name.show();
        user_profile_dialog.handleUpdate();
      });

      function generate_webauthn_list_item_html(credential) {
        item_html = `
        <li class="mdui-list-item mdui-ripple">
          <i class="mdui-list-item-icon mdui-icon material-icons">fingerprint</i>
          <div class="mdui-list-item-content">
          <div>
          ${credential.name}
        `;
        if (credential.passkey) {
          item_html += `
            <div class="mdui-chip mdui-m-l-1">
              <span class="mdui-chip-title">Passkey</span>
            </div>
          `;
        }
        let created_at = new Date(credential.created_at);
        item_html += `
        </div>
          <div class="mdui-text-right mdui-text-color-gray">
            <small>${created_at.toISOString().substring(0, 10)}</small>
          </div>
          </div>
        `;
        item_html += `
          <i class="mdui-list-item-icon mdui-icon material-icons user-profile-webauthn-manage-del">delete</i>
        </li>
        `;
        return item_html;
      }

      function register_action_button(item, credential) {
        item.find(".user-profile-webauthn-manage-del").on("click", function () {
          $.ajax({
            method: "POST",
            url: "api/user/webauthn/delete",
            data: JSON.stringify([credential.name]),
            contentType: "application/json",
            processData: false,
            complete: function (xhr) {
              let response = JSON.parse(xhr.responseText || "{}");
              if (xhr.status == 200 && response.code === 0) {
                item.remove();
                user_profile_dialog.handleUpdate();
              } else {
                mdui.snackbar("删除失败: " + response.error);
              }
            }
          });
        });
      }

      function update_webauthn_list() {
        user_profile_dialog_progress.show();
        $.ajax({
          method: "GET",
          url: "api/user/webauthn/list",
          headers: {
            Accept: "application/json"
          },
          complete: function (xhr) {
            let response = JSON.parse(xhr.responseText || "{}");
            if (xhr.status == 200 && response.code === 0) {
              user_profile_webauthn_manage_list.empty();
              for (let credential of response.credentials) {
                let item = $(generate_webauthn_list_item_html(credential));
                user_profile_webauthn_manage_list.append(item);
                register_action_button(item, credential);
              }
            }
            user_profile_dialog_progress.hide();
            user_profile_dialog.handleUpdate();
          }
        });
      }

      function webauthn_back_to_main() {
        user_profile_webauthn_manage_name.hide();
        user_profile_webauthn_manage_main.show();
        update_webauthn_list();
        user_profile_dialog.handleUpdate();
      }

      user_profile_webauthn_manage_name_cancel.on("click", webauthn_back_to_main);

      user_profile_webauthn_manage_name_confirm.on("click", function () {
        let name = user_profile_webauthn_manage_name_input.val();
        if (name.length == 0) {
          mdui.snackbar("请输入名称");
          return;
        }

        if (!navigator.credentials) {
          mdui.snackbar("WebAuthn 不可用");
          return;
        }

        user_profile_webauthn_manage_action.attr("disabled", "disabled");
        user_profile_webauthn_manage_name_row.hide();
        user_profile_webauthn_manage_name_loading.show();
        user_profile_dialog_progress.show();
        user_profile_dialog.handleUpdate();

        function post_register() {
          user_profile_webauthn_manage_action.removeAttr("disabled");
          user_profile_webauthn_manage_name_row.show();
          user_profile_webauthn_manage_name_loading.hide();
          user_profile_dialog_progress.hide();
          user_profile_dialog.handleUpdate();
          webauthn_back_to_main();
        }

        $.ajax({
          method: "POST",
          url: "api/user/webauthn/register/request",
          data: JSON.stringify({
            name: name,
            passkey: user_profile_webauthn_manage_register_as_passkey.prop("checked")
          }),
          contentType: "application/json",
          processData: false,
          complete: function (xhr) {
            if (xhr.status == 200) {
              let response = JSON.parse(xhr.responseText || "{}");
              if (response.code === 0) {
                response.publicKey.challenge = bufferDecode(response.publicKey.challenge);
                response.publicKey.user.id = bufferDecode(response.publicKey.user.id);
                if (response.publicKey.excludeCredentials && response.publicKey.excludeCredentials.length > 0) {
                  response.publicKey.excludeCredentials = response.publicKey.excludeCredentials.map(credential => {
                    credential.id = bufferDecode(credential.id);
                    return credential;
                  });
                }
                navigator.credentials
                  .create({
                    publicKey: response.publicKey
                  })
                  .then(credential => {
                    $.ajax({
                      method: "POST",
                      url: "api/user/webauthn/register",
                      headers: {
                        "X-WebAuthn-Session": response.session
                      },
                      data: JSON.stringify({
                        id: credential.id,
                        rawId: bufferEncode(credential.rawId),
                        type: credential.type,
                        response: {
                          attestationObject: bufferEncode(credential.response.attestationObject),
                          clientDataJSON: bufferEncode(credential.response.clientDataJSON)
                        }
                      }),
                      contentType: "application/json",
                      processData: false,
                      complete: function (xhr) {
                        let response = JSON.parse(xhr.responseText || "{}");
                        if (xhr.status == 200 && response.code === 0) {
                          mdui.snackbar("注册成功");
                        } else {
                          mdui.snackbar("注册失败: " + response.error);
                        }
                        post_register();
                      }
                    });
                  })
                  .catch(err => {
                    mdui.snackbar("注册失败: " + err);
                    post_register();
                  });
              } else {
                mdui.snackbar("注册失败: " + response.error);
                post_register();
              }
            } else {
              mdui.snackbar("网络错误");
              post_register();
            }
          }
        });
      });
    })();

    const pasteSearchInput = $("#paste-search-input");
    const pasteSearchResults = $("#paste-search-results");

    let firstSearchDone = false;
    let debounceTimer = null;
    
    pasteSearchInput.on("input", function () {
      const query = pasteSearchInput.val().trim();
      clearTimeout(debounceTimer);
    
      if (query === "") {
        pasteSearchResults.empty();
        return;
      }
    
      if (!firstSearchDone) {
        firstSearchDone = true;
        doSearch(query);
      } else {
        debounceTimer = setTimeout(() => {
          doSearch(query);
        }, 500);
      }
    });
    
    function doSearch(query) {
      $.ajax({
        method: "GET",
        url: "/api/user/search",
        data: { title: query },
        success: function (response) {
          response = JSON.parse(response || "{}");
          if (response.code === 0) {
            displaySearchResults(response.results);
          } else {
            mdui.snackbar({ message: "搜索失败: " + response.message, position: "top" });
          }
        },
        error: function () {
          mdui.snackbar({ message: "搜索请求失败", position: "top" });
        }
      });
    }

    function displaySearchResults(pastes) {
      pasteSearchResults.empty();
      if (pastes.length === 0) {
        pasteSearchResults.append('<div class="mdui-list-item">没有找到相关结果</div>');
        return;
      }
      pastes.forEach(paste => {
        const pasteItem = `
          <a href="/#${paste.short_url}" class="mdui-list-item">
            <div class="mdui-list-item-content">
              <div class="mdui-list-item-title">${paste.filename}</div>
              <div class="mdui-list-item-text">创建于: ${new Date(paste.created_at).toLocaleString()}</div>
              <div class="mdui-list-item-text">大小: ${paste.size}</div>
            </div>
          </a>
        `;
        pasteSearchResults.append(pasteItem);
      });
    }

  });
})();
