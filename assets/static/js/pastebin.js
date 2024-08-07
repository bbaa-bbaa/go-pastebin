(function () {
  let query_hash = window.location.hash.replace(/^#/, "");
  let $ = mdui.$;
  const well_known_text_mime_types = [
    "text/plain",
    "text/html",
    "text/css",
    "text/javascript",
    "application/json",
    "application/xml",
    "application/xhtml+xml",
    "application/rss+xml",
    "application/atom+xml",
    "application/mathml+xml",
    "application/ecmascript",
    "application/x-javascript",
    "application/x-latex",
    "application/x-markdown",
    "application/x-yaml"
  ];
  function easeInSine(t) {
    return 1 - Math.cos((t * Math.PI) / 2);
  }
  function isDesktop() {
    return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) === false;
  }
  $(function () {
    document.body.addEventListener("drop", function (e) {
      e.preventDefault();
      e.stopPropagation();
    });
    document.body.addEventListener("dragover", function (e) {
      e.preventDefault();
    });
    function Collapse(jq, heightBox, margin) {
      this.$ = jq;
      this.element = (heightBox || jq).get(0);
      this.expand = this.$.hasClass("card-collapse-open");
      this.autoLock = false;
      this.margin = margin === undefined ? 16 : margin;
      this.callback = null;
      this.$.on("transitionend", e => {
        if (!heightBox && !this.autoLock && this.expand) {
          this.$.css("height", "auto");
        }
        if (this.callback) {
          this.callback();
          this.callback = null;
        }
      });
    }

    Collapse.prototype.fixed = function () {
      if (this.expand) {
        this.$.css("height", this.element.scrollHeight + "px");
      }
    };

    Collapse.prototype.close = function (fixed) {
      this.autoLock = true;
      if (!fixed) {
        this.fixed();
        requestAnimationFrame(() => {
          this.close(true);
        });
        return new Promise(resolve => {
          this.callback = resolve;
        });
      }
      this.expand = false;
      this.autoLock = false;
      this.$.css("height", "0");
      if (this.margin) {
        this.$.css("margin", "0");
      }
    };

    Collapse.prototype.open = function () {
      this.expand = true;
      this.$.css("height", this.element.scrollHeight + "px");
      if (this.margin) {
        this.$.css("margin", this.margin + "px 0");
      }
      return new Promise(resolve => {
        this.callback = resolve;
      });
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

    const paste_mdui_tab = new mdui.Tab("#paste-mdui-tab");
    (function new_paste() {
      const text_input = $("#new-paste-text-input");
      const file_input = $("#new-paste-file-input");
      const file_paste = $("#new-paste-file");
      const drop_file_overlay = $(".paste-file-drop-overlay");
      const paste_password = $("#new-paste-password");
      const paste_expire = $("#new-paste-expire");
      const paste_max_access_count = $("#new-paste-max-access-count");
      const paste_uuid = $("#new-paste-uuid");
      const paste_short_url = $("#new-paste-short-url");
      const paste_delete_if_expired = $("#new-paste-delete-if-expired");
      const paste_delete = $("#new-paste-delete");
      const paste_update = $("#new-paste-update");
      const paste_submit = $("#new-paste-submit");
      const paste_load = $("#new-paste-load-from-file");

      const file_paste_icon = $("#new-paste-file-icon");
      const file_paste_preview = $("#new-paste-file-preview");
      const file_paste_filename = $("#new-paste-file-filename");
      const file_paste_progress = $("#new-paste-file-progress");
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

      let text_file = {
        filename: "",
        mime_type: ""
      };
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

      function show_file_paste_info() {
        if (!paste_file) {
          return;
        }
        file_paste_filename.text(paste_file.name + " (" + (paste_file.size / 1024 / 1024).toFixed(2).toString() + " MiB)");
        paste_preview(paste_file);
        paste_load.text("切换到文本模式").removeClass("mdui-color-theme-accent").addClass("mdui-color-blue-accent");
        text_input.parent().hide();
        file_paste.show();
      }

      function set_paste_file(file) {
        if (!file) {
          return;
        }
        // 小于 32kb 的文本文件，直接读取内容
        if (file.size <= 32 * 1024 && well_known_text_mime_types.some(type => new RegExp(type).test(file.type))) {
          const reader = new FileReader();
          reader.onload = function () {
            text_input.val(reader.result);
            text_input.get(0).dispatchEvent(new Event("input"));
          };
          reader.readAsText(file);
          paste_file = null;
          text_file.filename = file.name;
          text_file.mime_type = file.type;
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
        text_input.parent().show();
      }

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
        } else {
          paste_delete.attr("disabled", "disabled");
          paste_update.attr("disabled", "disabled");
        }
      });

      function check_short_url(url) {
        return url.length == 0 || /^[A-Za-z0-9\\._-]+$/im.test(url);
      }

      const check_short_url_available = _.debounce(function (url) {
        $.ajax({
          method: "GET",
          url: "api/check_url/" + url
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

      function check_allow_delete_if_expired() {
        if (paste_expire.val() != "0" || parseInt(paste_max_access_count.val()) > 0) {
          paste_delete_if_expired.removeAttr("disabled");
        } else {
          paste_delete_if_expired.attr("disabled", "disabled");
          paste_delete_if_expired.prop("checked", false);
        }
      }

      paste_expire.on("change", check_allow_delete_if_expired);
      paste_max_access_count.on("input", check_allow_delete_if_expired);

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
          QRCode.toCanvas(new_paste_result_qr_code.get(0), response.url, { margin: 0, scale: 6 }, function () { });
          new_paste_result_link.closest("div").show();
          new_paste_result_qr_code.show();
        } else {
          new_paste_result_link.closest("div").hide();
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
        result_collapse.close();
      }

      function upload_progress(e) {
        if (e.lengthComputable) {
          file_paste_progress_text.text(
            (e.loaded / 1024 / 1024).toFixed(2) + " MiB / " + (e.total / 1024 / 1024).toFixed(2) + " MiB - " + ((e.loaded / e.total) * 100).toFixed(2) + "%"
          );
          file_paste_progress_bar.css("width", ((e.loaded / e.total) * 100).toFixed(2) + "%");
        }
      }

      function prepare_data() {
        let text = text_input.val();
        let password = paste_password.val();
        let expire = paste_expire.val();
        let max_access_count = paste_max_access_count.val();
        let short_url = paste_short_url.val();
        let delete_if_expired = paste_delete_if_expired.prop("checked");
        let data = new FormData();
        let query_params = {};
        if (password.length != 0) {
          query_params.password = password;
        }

        if (expire != "0") {
          query_params.expire = new Date().getTime() + parseInt(expire);
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

        if (delete_if_expired) {
          query_params.delete_if_expired = "true";
        }

        if (paste_file) {
          data.append("c", paste_file);
        } else {
          data.append("c", new File([text], text_file.filename || "-", { type: text_file.mime_type == "" ? "text/plain; charset=utf-8" : text_file.mime_type }));
        }
        return { data, query_params };
      }

      paste_submit.on("click", function () {
        let prepared_data = prepare_data();
        let data = prepared_data.data;
        let query_params = prepared_data.query_params;
        action_button.attr("disabled", "disabled");
        hide_result();
        const query_string = $.param(query_params).trim();
        $.ajax({
          method: "POST",
          url: "/" + query_string !== "" ? "?" + query_string : "",
          data: data,
          headers: {
            Accept: "application/json"
          },
          contentType: false,
          processData: false,
          beforeSend: function (xhr) {
            if (paste_file) {
              upload_progress({ loaded: 0, total: paste_file.size, lengthComputable: true });
              xhr.upload.addEventListener("progress", upload_progress);
              file_paste_progress.css("height", "18px");
            }
          }
        })
          .then(res => {
            let response = JSON.parse(res);
            if (response.code != 0) {
              paste_submit.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
              setTimeout(() => {
                paste_submit.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
              }, 1000);
            } else {
              paste_submit.removeClass("mdui-color-theme-accent").addClass("mdui-color-green-accent");
              setTimeout(() => {
                paste_submit.removeClass("mdui-color-green-accent").addClass("mdui-color-theme-accent");
              }, 1000);
            }
            return show_result("创建结果", response, false);
          })
          .catch(res => {
            let response = JSON.parse(res);
            paste_submit.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
            setTimeout(() => {
              paste_submit.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
            }, 1000);
            mdui.snackbar("创建失败: " + response.error);
          })
          .finally(() => {
            action_button.removeAttr("disabled");
            file_paste_progress.css("height", "0px");
          });
      });

      paste_update.on("click", function () {
        let uuid = paste_uuid.val();
        if (!check_uuid(uuid) || uuid.length == 0) {
          mdui.snackbar("无效的 UUID");
          return;
        }
        let prepared_data = prepare_data();
        let data = prepared_data.data;
        if (data.get("c").size == 0) {
          data.delete("c");
        }
        let query_params = prepared_data.query_params;

        action_button.attr("disabled", "disabled");
        hide_result();
        const query_string = $.param(query_params).trim();
        $.ajax({
          method: "PUT",
          url: "/" + uuid + (query_string !== "" ? "?" + query_string : ""),
          data: data,
          headers: {
            Accept: "application/json"
          },
          contentType: false,
          processData: false,
          beforeSend: function (xhr) {
            if (paste_file) {
              upload_progress({ loaded: 0, total: paste_file.size, lengthComputable: true });
              xhr.upload.addEventListener("progress", upload_progress);
              file_paste_progress.css("height", "18px");
            }
          }
        })
          .then(res => {
            let response = JSON.parse(res);
            if (response.code != 0) {
              paste_update.removeClass("mdui-color-blue-accent").addClass("mdui-color-red-accent");
              setTimeout(() => {
                paste_update.removeClass("mdui-color-red-accent").addClass("mdui-color-blue-accent");
              }, 1000);
            } else {
              paste_update.removeClass("mdui-color-blue-accent").addClass("mdui-color-green-accent");
              setTimeout(() => {
                paste_update.removeClass("mdui-color-green-accent").addClass("mdui-color-blue-accent");
              }, 1000);
            }
            return show_result("更新结果", response, false);
          })
          .catch(res => {
            let response = JSON.parse(res);
            paste_update.removeClass("mdui-color-blue-accent").addClass("mdui-color-red-accent");
            setTimeout(() => {
              paste_update.removeClass("mdui-color-red-accent").addClass("mdui-color-blue-accent");
            }, 1000);
            mdui.snackbar("删除失败: " + response.error);
          })
          .finally(() => {
            action_button.removeAttr("disabled");
            file_paste_progress.css("height", "0px");
          });
      });

      paste_delete.on("click", function () {
        let uuid = paste_uuid.val();
        if (!check_uuid(uuid) || uuid.length == 0) {
          mdui.snackbar("无效的 UUID");
          return;
        }

        action_button.attr("disabled", "disabled");
        $.ajax({
          method: "DELETE",
          url: "/" + uuid,
          headers: {
            Accept: "application/json"
          },
          contentType: false,
          processData: false
        })
          .then(res => {
            let response = JSON.parse(res);
            if (response.code != 0) {
              paste_delete.removeClass("mdui-color-red").addClass("mdui-color-red-800");
              setTimeout(() => {
                paste_delete.removeClass("mdui-color-red-800").addClass("mdui-color-red");
              }, 1000);
            } else {
              paste_delete.removeClass("mdui-color-red").addClass("mdui-color-green-accent");
              setTimeout(() => {
                paste_delete.removeClass("mdui-color-green-accent").addClass("mdui-color-red");
              }, 1000);
            }
            return show_result("删除结果", response, true);
          })
          .catch(res => {
            let response = JSON.parse(res);
            paste_delete.removeClass("mdui-color-red").addClass("mdui-color-red-800");
            setTimeout(() => {
              paste_delete.removeClass("mdui-color-red-800").addClass("mdui-color-red");
            }, 1000);
            mdui.snackbar("删除失败: " + response.error);
          })
          .finally(() => {
            action_button.removeAttr("disabled");
            file_paste_progress.css("height", "0px");
          });
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
    })();
    (function paste_viewer() {
      const paste_viewer_query = $("#paste-viewer-query");
      const paste_viewer_password = $("#paste-viewer-password");
      const paste_viewer_text = $("#paste-viewer-text");

      const paste_viewer_not_found = $("#paste-viewer-not-found");
      const paste_viewer_file = $("#paste-viewer-file");
      const collapse_manager = new CollapseGroup({
        paste_viewer_query,
        paste_viewer_password,
        paste_viewer_not_found,
        paste_viewer_text,
        paste_viewer_file
      });

      const paste_viewer_back_to_query = $(".paste-viewer-back-to-query");
      const paste_viewer_action = $(".paste-viewer-action");

      const paste_viewer_query_btn = $("#paste-viewer-query-btn");
      const paste_viewer_query_input = $("#paste-viewer-query-input");
      const paste_viewer_title = $(".paste-viewer-title");

      const paste_viewer_password_input = $("#paste-viewer-password-input");
      const paste_viewer_confirm_password = $("#paste-viewer-confirm-password");

      const paste_viewer_text_content_wrapper = $("#paste-viewer-text-content-wrapper");
      const paste_viewer_text_content = paste_viewer_text_content_wrapper.children("div");
      const paste_viewer_enable_markdown_render = $("#paste-viewer-enable-markdown-render");

      const paste_viewer_download_btn = $(".paste-viewer-download-btn");

      const paste_viewer_file_icon = $("#paste-viewer-file-icon");
      const paste_viewer_file_preview = $("#paste-viewer-file-preview");
      const paste_viewer_file_filename = $("#paste-viewer-file-filename");

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
        let timeout = setTimeout(() => {
          timeout = null;
          collapse_manager.paste_viewer_file.open();
          paste_viewer_action.removeAttr("disabled");
        }, 1000);
        let start = new Date().getTime();
        return function show() {
          if (timeout) {
            clearTimeout(timeout);
            let end = new Date().getTime()
            if (end - start < 300) {
              setTimeout(() => {
                collapse_manager.paste_viewer_file.open();
              }, 300 - (end - start));
            } else {
              collapse_manager.paste_viewer_file.open();
            }
          }
        }
      }
      function paste_preview_file() {
        paste_viewer_file_filename.text(paste_metadata.filename + " (" + (paste_metadata.size / 1024 / 1024).toFixed(2).toString() + " MiB)");
        paste_viewer_file_icon.hide();
        paste_viewer_file_preview.show();
        let show = paste_preview_file_show();
        if (paste_metadata.type.startsWith("image/")) {
          $(`<img style="max-height: inherit; max-width:100%">`).on("load", show).attr("src", paste_metadata.url).appendTo(paste_viewer_file_preview);
        } else if (paste_metadata.type.startsWith("audio/")) {
          $('<audio controls style="max-height: inherit; max-width:100%">').on("loadedmetadata", show).attr("src", paste_metadata.url).appendTo(paste_viewer_file_preview);
        } else if (paste_metadata.type.startsWith("video/")) {
          $('<video controls style="max-height: inherit; max-width:100%">').on("loadedmetadata", show).attr("src", paste_metadata.url).appendTo(paste_viewer_file_preview);
        } else {
          paste_viewer_file_preview.hide();
          paste_viewer_file_icon.show();
        }

      }
      let collapse_paste_viewer_text_content = new Collapse(paste_viewer_text_content_wrapper, paste_viewer_text_content, 0);
      function paste_preview_text_render(init) {
        if (paste_viewer_enable_markdown_render.prop("checked")) {
          paste_viewer_text_content.css("white-space", "");
          paste_viewer_text_content.html(DOMPurify.sanitize(marked.parse(paste_metadata.content)));
        } else {
          paste_viewer_text_content.css("white-space", "pre-wrap");
          paste_viewer_text_content.text(paste_metadata.content);
          collapse_manager.paste_viewer_text.open();
        }
        if (init) {
          paste_viewer_text_content_wrapper.css("height", "auto")
          collapse_manager.paste_viewer_text.open();
        }
        collapse_paste_viewer_text_content.open();
      }

      paste_viewer_enable_markdown_render.on("change", () => { paste_preview_text_render() });

      function paste_preview_text() {
        $.ajax({
          method: "GET",
          url: "/" + paste_metadata.id //+ "?access_token=" + paste_metadata.access_token,
        })
          .then(res => {
            paste_metadata.content = res;
            if (paste_metadata.type.startsWith("text/markdown")) {
              paste_viewer_enable_markdown_render.prop("checked", true);
            }
            paste_preview_text_render(true);
          })
          .catch(() => {
            paste_viewer_text_content.text("无法加载 Paste");
            collapse_paste_viewer_text_content.open();
          })
      }

      function paste_preview() {
        if (paste_metadata.type.startsWith("text/") && paste_metadata.size <= 1024 * 1024) {
          paste_preview_text();
          paste_viewer_action.removeAttr("disabled");
        } else {
          paste_preview_file();
        }
      }
      function parse_filename(xhr) {
        try {
          if (TextDecoder) {
            let filename = xhr.getResponseHeader("X-Origin-Filename").split("").map(a => a.charCodeAt(0));
            let utf8_decoder = new TextDecoder("utf-8");
            return utf8_decoder.decode(new Uint8Array(filename));
          }
        } catch (e) {
        }
        let urlencode_filename = xhr.getResponseHeader("X-Origin-Filename-Encoded");
        return decodeURIComponent(urlencode_filename);
      }
      function query_paste_metadata(id, password) {
        paste_viewer_action.attr("disabled", "disabled");
        paste_viewer_title.text("Paste: " + id);
        query_id = id;
        $.ajax({
          method: "HEAD",
          url: "/" + id + (password ? "?pwd=" + password : ""),
          complete: function (xhr) {
            if (xhr.status == 200) {
              paste_metadata = {
                id: id,
                password: password,
                size: xhr.getResponseHeader("Content-Length"),
                type: xhr.getResponseHeader("Content-Type"),
                filename: parse_filename(xhr),
                access_token: xhr.getResponseHeader("X-Access-Token"),
                url: "/" + id + "?access_token=" + xhr.getResponseHeader("X-Access-Token")
              };
              paste_viewer_download_btn.attr("download", paste_metadata.filename).attr("href", paste_metadata.url);
              paste_preview();
            } else if (xhr.status == 404) {
              collapse_manager.paste_viewer_not_found.open();
              paste_viewer_action.removeAttr("disabled");
            } else if (xhr.status == 401) {
              if (!password) {
                collapse_manager.paste_viewer_password.open();
              } else {
                paste_viewer_confirm_password.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
                setTimeout(() => {
                  paste_viewer_confirm_password.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
                }, 1000);
                mdui.snackbar("密码错误");
              }
              paste_viewer_action.removeAttr("disabled");
            } else {
              paste_viewer_action.removeAttr("disabled");
            }
          }
        }).catch(() => { });
      }

      paste_viewer_query_btn.on("click", function () {
        query_id = paste_viewer_query_input.val();
        if (query_id.length == 0) {
          mdui.snackbar("请输入 Paste Short URL 或 Paste Hash");
          return;
        }
        query_id = query_id.split("/").pop().replace(/^#/m, "");
        query_paste_metadata(query_id);
      });

      paste_viewer_back_to_query.on("click", function () {
        collapse_manager.paste_viewer_query.open().then(() => {
          paste_viewer_file_preview.html("");
          paste_viewer_file_icon.show();
          paste_viewer_text_content.html("");
          collapse_paste_viewer_text_content.open();
        });
      });

      paste_viewer_confirm_password.on("click", function () {
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
        if (password) {
          paste_viewer_password_input.val(password);
          paste_viewer_password_input.get(0).dispatchEvent(new Event("input"));
        }
        paste_mdui_tab.show(1);
        paste_viewer_query_input.val(query_hash);
        paste_viewer_query_input.get(0).dispatchEvent(new Event("input"));
        query_paste_metadata(query_hash);
      }
    })();
  });
})();
